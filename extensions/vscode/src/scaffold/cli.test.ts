// Unit tests for the CLI wrapper. They run under `node --test`, with no VS Code
// host and without spawning any process: `exec` is injected.
import * as assert from "node:assert/strict";
import * as path from "node:path";
import { test } from "node:test";

import {
	BinaryNotFoundError,
	Exec,
	GenerationError,
	binaryCandidates,
	buildInitArgs,
	goBinDirs,
	isOurCli,
	locateBinary,
	parseGenerationOutput,
	parseVersion,
	runInit,
} from "./cli";

const NO_ENV: NodeJS.ProcessEnv = {};
const HOME = path.join(path.sep, "home", "me");

test("parseGenerationOutput reads created and skipped files", () => {
	const output = [
		"Create Dockerfile",
		"Create cmd/main.go",
		"Skip   Makefile (already exists, use --force to overwrite)",
		"Skip   config/config.yml (already exists, use --force to overwrite)",
		"",
		"2 file(s) created in /tmp/svc, 2 skipped (re-run with --force to overwrite)",
		"Next: go mod tidy && make run",
	].join("\n");

	const result = parseGenerationOutput(output);

	assert.deepEqual(result.created, ["Dockerfile", "cmd/main.go"]);
	// The "(already exists…)" hint must not end up in the path.
	assert.deepEqual(result.skipped, ["Makefile", "config/config.yml"]);
});

test("parseGenerationOutput ignores the summary lines", () => {
	// "Next: …" and the count line must not be mistaken for files.
	const result = parseGenerationOutput(
		"13 file(s) created in /tmp/svc\nNext: go mod tidy && make run\n",
	);

	assert.deepEqual(result, { created: [], skipped: [] });
});

test("parseGenerationOutput tolerates CRLF", () => {
	const result = parseGenerationOutput("Create go.mod\r\nCreate cmd/main.go\r\n");

	assert.deepEqual(result.created, ["go.mod", "cmd/main.go"]);
});

test("buildInitArgs always passes the directory explicitly", () => {
	// Without it the CLI would generate into its own cwd, not the picked folder.
	assert.deepEqual(buildInitArgs("/tmp/svc"), ["init", "/tmp/svc"]);
	assert.deepEqual(buildInitArgs("/tmp/svc", { force: true }), [
		"init",
		"--force",
		"/tmp/svc",
	]);
});

test("goBinDirs prefers GOBIN, then every GOPATH entry, then ~/go/bin", () => {
	const dirs = goBinDirs(
		{
			GOBIN: path.join(path.sep, "gobin"),
			GOPATH: [path.join(path.sep, "ws", "a"), path.join(path.sep, "ws", "b")].join(
				path.delimiter,
			),
		},
		HOME,
	);

	assert.deepEqual(dirs, [
		path.join(path.sep, "gobin"),
		path.join(path.sep, "ws", "a", "bin"),
		path.join(path.sep, "ws", "b", "bin"),
		path.join(HOME, "go", "bin"),
	]);
});

test("goBinDirs falls back to ~/go/bin when GOPATH is unset", () => {
	assert.deepEqual(goBinDirs(NO_ENV, HOME), [path.join(HOME, "go", "bin")]);
});

test("binaryCandidates puts the configured path first and bare names before absolute ones", () => {
	const candidates = binaryCandidates("/opt/bin/my-scaffold", NO_ENV, HOME);

	assert.equal(candidates[0], "/opt/bin/my-scaffold");
	// Bare names next, so PATH resolution is left to the OS.
	assert.deepEqual(candidates.slice(1, 3), ["scaffold", "Go-DDD-Scaffold"]);
	// go install names the binary after the module, hence the second name.
	assert.ok(candidates.includes(path.join(HOME, "go", "bin", "Go-DDD-Scaffold")));
});

test("binaryCandidates has no duplicates", () => {
	const candidates = binaryCandidates("scaffold", NO_ENV, HOME);

	assert.equal(new Set(candidates).size, candidates.length);
});

test("isOurCli accepts our banner and rejects a foreign scaffold tool", () => {
	assert.ok(isOurCli("scaffold version v1.2.3\n"));
	assert.ok(!isOurCli("Scaffold 4.1.0 (Java project generator)\n"));
	assert.ok(!isOurCli(""));
});

test("parseVersion extracts the version, or reports it as unknown", () => {
	assert.equal(parseVersion("scaffold version v0.8.0\n"), "v0.8.0");
	assert.equal(parseVersion("scaffold version\n"), "unknown");
});

test("locateBinary skips a same-named program that is not ours", async () => {
	const probed: string[] = [];
	const exec: Exec = async (file) => {
		probed.push(file);
		if (file === "scaffold") {
			return "Scaffold 4.1.0 (some other generator)\n";
		}
		if (file === "Go-DDD-Scaffold") {
			return "scaffold version v0.8.0\n";
		}
		throw new Error("ENOENT");
	};

	const found = await locateBinary("", exec, NO_ENV, HOME);

	assert.equal(found.command, "Go-DDD-Scaffold");
	assert.equal(found.version, "v0.8.0");
	assert.deepEqual(probed, ["scaffold", "Go-DDD-Scaffold"]);
});

test("locateBinary falls back to ~/go/bin when the PATH does not carry it", async () => {
	// The usual case on macOS: a GUI editor never read the shell profile.
	const inGoBin = path.join(HOME, "go", "bin", "scaffold");
	const exec: Exec = async (file) => {
		if (file === inGoBin) {
			return "scaffold version v0.8.0\n";
		}
		throw new Error("ENOENT");
	};

	const found = await locateBinary("", exec, NO_ENV, HOME);

	assert.equal(found.command, inGoBin);
});

test("locateBinary reports every candidate it tried", async () => {
	const exec: Exec = async () => {
		throw new Error("ENOENT");
	};

	await assert.rejects(
		() => locateBinary("", exec, NO_ENV, HOME),
		(err: unknown) => {
			assert.ok(err instanceof BinaryNotFoundError);
			assert.ok(err.triedCandidates.includes("scaffold"));
			assert.match(err.message, /not found/);
			return true;
		},
	);
});

test("runInit returns the parsed result and the raw output", async () => {
	const exec: Exec = async (file, args) => {
		assert.equal(file, "scaffold");
		assert.deepEqual(args, ["init", "--force", "/tmp/svc"]);
		return "Create go.mod\nSkip   Makefile (already exists, use --force to overwrite)\n";
	};

	const { result, output } = await runInit(
		"scaffold",
		"/tmp/svc",
		{ force: true },
		exec,
	);

	assert.deepEqual(result, { created: ["go.mod"], skipped: ["Makefile"] });
	assert.match(output, /Create go\.mod/);
});

test("runInit wraps a failing CLI in a GenerationError that keeps the output", async () => {
	const exec: Exec = async () => {
		throw new Error("exit status 1: permission denied");
	};

	await assert.rejects(
		() => runInit("scaffold", "/root/nope", {}, exec),
		(err: unknown) => {
			assert.ok(err instanceof GenerationError);
			assert.match(err.output, /permission denied/);
			return true;
		},
	);
});
