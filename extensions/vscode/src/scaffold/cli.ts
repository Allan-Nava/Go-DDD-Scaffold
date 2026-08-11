// Thin wrapper around the `scaffold` CLI.
//
// This module deliberately does NOT import "vscode": the extension delegates
// every generation decision to the binary, so the templates live in exactly one
// place (template/ + static/ in the CLI, embedded in it). Keeping this file free
// of the editor API is also what makes it unit-testable outside a VS Code host.
import { execFile } from "node:child_process";
import * as os from "node:os";
import * as path from "node:path";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

/**
 * Names the CLI can have on disk. `go install` names the binary after the
 * module, so it lands as "Go-DDD-Scaffold"; the README suggests renaming it to
 * "scaffold", and the release archives ship it under that name.
 */
export const BINARY_NAMES = ["scaffold", "Go-DDD-Scaffold"] as const;

/** What `scaffold --version` starts with. Used to tell our CLI from any other
 * program that happens to be called "scaffold" on the user's PATH. */
export const VERSION_PREFIX = "scaffold version";

/** How long a candidate binary gets to answer `--version`. */
const PROBE_TIMEOUT_MS = 5_000;

/** How long a generation gets. Writing ~13 small files is milliseconds; this is
 * only here so a wedged process cannot hang the extension host forever. */
const GENERATE_TIMEOUT_MS = 60_000;

export interface GenerationResult {
	/** Destination-relative paths the CLI created. */
	created: string[];
	/** Paths left untouched because they already existed. */
	skipped: string[];
}

/** Raised when no usable CLI could be found, so the caller can offer to install it. */
export class BinaryNotFoundError extends Error {
	constructor(readonly triedCandidates: string[]) {
		super(
			"the scaffold CLI was not found. Tried: " + triedCandidates.join(", "),
		);
		this.name = "BinaryNotFoundError";
	}
}

/** Raised when the CLI ran but reported a failure. */
export class GenerationError extends Error {
	constructor(message: string, readonly output: string) {
		super(message);
		this.name = "GenerationError";
	}
}

/**
 * goBinDirs lists the directories `go install` writes binaries to. They are
 * often absent from the PATH the editor inherits (a GUI app on macOS does not
 * read the shell profile), which is the usual reason a working CLI looks
 * missing from inside VS Code.
 */
export function goBinDirs(
	env: NodeJS.ProcessEnv = process.env,
	homedir: string = os.homedir(),
): string[] {
	const dirs: string[] = [];
	if (env.GOBIN) {
		dirs.push(env.GOBIN);
	}
	// GOPATH may hold several entries, like PATH.
	for (const gopath of (env.GOPATH ?? "").split(path.delimiter)) {
		if (gopath) {
			dirs.push(path.join(gopath, "bin"));
		}
	}
	dirs.push(path.join(homedir, "go", "bin")); // the default when GOPATH is unset
	return [...new Set(dirs)];
}

/**
 * binaryCandidates is the ordered list of commands to probe. Bare names come
 * first so PATH resolution is left to the OS; the explicit go/bin paths are the
 * fallback for the inherited-PATH problem described in goBinDirs.
 */
export function binaryCandidates(
	configuredPath = "",
	env: NodeJS.ProcessEnv = process.env,
	homedir: string = os.homedir(),
): string[] {
	const candidates: string[] = [];
	const configured = configuredPath.trim();
	if (configured) {
		candidates.push(configured);
	}
	candidates.push(...BINARY_NAMES);
	for (const dir of goBinDirs(env, homedir)) {
		for (const name of BINARY_NAMES) {
			candidates.push(path.join(dir, name));
		}
	}
	return [...new Set(candidates)];
}

/** isOurCli reports whether `--version` output came from this project's CLI. */
export function isOurCli(versionOutput: string): boolean {
	return versionOutput.trimStart().startsWith(VERSION_PREFIX);
}

/** parseVersion extracts the version string from `scaffold --version` output. */
export function parseVersion(versionOutput: string): string {
	const line = versionOutput.trim().split("\n", 1)[0] ?? "";
	return line.slice(VERSION_PREFIX.length).trim() || "unknown";
}

/** buildInitArgs assembles the argv for a generation run. */
export function buildInitArgs(
	targetDir: string,
	options: { force?: boolean } = {},
): string[] {
	const args = ["init"];
	if (options.force) {
		args.push("--force");
	}
	// The directory is passed explicitly instead of relying on a working
	// directory: the CLI defaults to its own cwd, which is not the folder the
	// user picked in the explorer.
	args.push(targetDir);
	return args;
}

/**
 * parseGenerationOutput reads the CLI's per-file progress lines:
 *
 *     Create cmd/main.go
 *     Skip   Makefile (already exists, use --force to overwrite)
 *
 * The trailing summary lines are ignored: the counts are derived from the file
 * lists, so the two can never disagree.
 */
export function parseGenerationOutput(output: string): GenerationResult {
	const created: string[] = [];
	const skipped: string[] = [];

	for (const raw of output.split(/\r?\n/)) {
		const line = raw.trimEnd();
		const match = /^(Create|Skip)\s+(.+)$/.exec(line);
		if (!match) {
			continue;
		}
		const [, verb, rest] = match;
		if (verb === "Create") {
			created.push(rest.trim());
		} else {
			// Drop the "(already exists, use --force to overwrite)" hint.
			skipped.push(rest.replace(/\s*\(already exists.*$/, "").trim());
		}
	}
	return { created, skipped };
}

/** Minimal seam over child_process, so the tests do not spawn anything. */
export type Exec = (
	file: string,
	args: string[],
	timeoutMs: number,
) => Promise<string>;

const defaultExec: Exec = async (file, args, timeoutMs) => {
	const { stdout, stderr } = await execFileAsync(file, args, {
		timeout: timeoutMs,
		// The CLI prints a handful of short lines; this is only a runaway guard.
		maxBuffer: 4 * 1024 * 1024,
	});
	return stdout + stderr;
};

/**
 * locateBinary probes the candidates and returns the first one that identifies
 * itself as this project's CLI. A candidate that cannot be executed, or that
 * answers with something else, is skipped rather than treated as an error: a
 * different program named "scaffold" on the PATH must not be run with `init`.
 */
export async function locateBinary(
	configuredPath = "",
	exec: Exec = defaultExec,
	env: NodeJS.ProcessEnv = process.env,
	homedir: string = os.homedir(),
): Promise<{ command: string; version: string }> {
	const candidates = binaryCandidates(configuredPath, env, homedir);

	for (const command of candidates) {
		let output: string;
		try {
			output = await exec(command, ["--version"], PROBE_TIMEOUT_MS);
		} catch {
			continue; // not present, not executable, or it timed out
		}
		if (isOurCli(output)) {
			return { command, version: parseVersion(output) };
		}
	}
	throw new BinaryNotFoundError(candidates);
}

/** runInit generates a project into targetDir and reports what changed. */
export async function runInit(
	command: string,
	targetDir: string,
	options: { force?: boolean } = {},
	exec: Exec = defaultExec,
): Promise<{ result: GenerationResult; output: string }> {
	const args = buildInitArgs(targetDir, options);

	let output: string;
	try {
		output = await exec(command, args, GENERATE_TIMEOUT_MS);
	} catch (err) {
		const detail = err instanceof Error ? err.message : String(err);
		throw new GenerationError(`\`${command} ${args.join(" ")}\` failed`, detail);
	}

	return { result: parseGenerationOutput(output), output };
}
