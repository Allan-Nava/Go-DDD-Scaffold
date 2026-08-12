// The "New Scaffold Project" command: pick a folder, run the CLI, report.
import { promises as fs } from "node:fs";
import * as vscode from "vscode";

import {
	BinaryNotFoundError,
	GenerationError,
	GenerationResult,
	OutdatedBinaryError,
	locateBinary,
	runInit,
} from "../scaffold/cli";

const CONFIG_SECTION = "goDddScaffold";
const INSTALL_COMMAND = "go install github.com/Allan-Nava/Go-DDD-Scaffold@latest";

/** newScaffoldProject is the handler registered for the command. `uri` is set
 * when the command comes from the explorer context menu. */
export async function newScaffoldProject(
	output: vscode.OutputChannel,
	uri?: vscode.Uri,
): Promise<void> {
	const targetDir = await resolveTargetDir(uri);
	if (!targetDir) {
		return; // the user cancelled
	}

	const force = await askAboutExistingFiles(targetDir);
	if (force === undefined) {
		return;
	}

	const configured = vscode.workspace
		.getConfiguration(CONFIG_SECTION)
		.get<string>("binaryPath", "");

	try {
		const { command, version } = await locateBinary(configured);
		output.appendLine(`> ${command} (${version})`);

		const { result, output: cliOutput } = await vscode.window.withProgress(
			{
				location: vscode.ProgressLocation.Notification,
				title: "Generating the Go DDD project layout…",
				cancellable: false,
			},
			() => runInit(command, targetDir, { force }),
		);

		output.append(cliOutput);
		await reportSuccess(result, targetDir, output);
	} catch (err) {
		await reportFailure(err, output);
	}
}

/** resolveTargetDir picks the destination: the folder from the context menu, the
 * single workspace folder, or one chosen from a dialog. */
async function resolveTargetDir(uri?: vscode.Uri): Promise<string | undefined> {
	if (uri && (await isDirectory(uri.fsPath))) {
		return uri.fsPath;
	}

	const folders = vscode.workspace.workspaceFolders ?? [];
	if (folders.length === 1) {
		return folders[0].uri.fsPath;
	}
	if (folders.length > 1) {
		const picked = await vscode.window.showWorkspaceFolderPick({
			placeHolder: "Where should the project be generated?",
		});
		return picked?.uri.fsPath;
	}

	const chosen = await vscode.window.showOpenDialog({
		canSelectFiles: false,
		canSelectFolders: true,
		canSelectMany: false,
		openLabel: "Generate here",
		title: "Select the folder for the new Go DDD project",
	});
	return chosen?.[0]?.fsPath;
}

/**
 * askAboutExistingFiles returns the value for --force, or undefined to cancel.
 * An empty folder needs no question. The CLI never overwrites without --force,
 * so the risky answer is the one the user has to pick explicitly.
 */
async function askAboutExistingFiles(
	targetDir: string,
): Promise<boolean | undefined> {
	let entries: string[];
	try {
		entries = await fs.readdir(targetDir);
	} catch {
		return false; // the CLI will create the directory, nothing to overwrite
	}
	if (entries.length === 0) {
		return false;
	}

	const skip = "Keep existing files";
	const overwrite = "Overwrite existing files";
	const picked = await vscode.window.showWarningMessage(
		`${vscode.workspace.asRelativePath(targetDir)} is not empty.`,
		{
			modal: true,
			detail:
				`"${skip}" generates only what is missing. ` +
				`"${overwrite}" replaces files such as cmd/main.go and go.mod with the template versions.`,
		},
		skip,
		overwrite,
	);
	if (picked === skip) {
		return false;
	}
	if (picked === overwrite) {
		return true;
	}
	return undefined;
}

async function reportSuccess(
	result: GenerationResult,
	targetDir: string,
	output: vscode.OutputChannel,
): Promise<void> {
	const { created, skipped } = result;

	if (created.length === 0 && skipped.length > 0) {
		void vscode.window.showInformationMessage(
			`Nothing to do: all ${skipped.length} file(s) already exist. ` +
				"Re-run and choose \"Overwrite existing files\" to replace them.",
		);
		return;
	}

	const parts = [`${created.length} file(s) created`];
	if (skipped.length > 0) {
		parts.push(`${skipped.length} kept`);
	}

	const showOutput = "Show Output";
	const openReadme = "Open README";
	const actions = [showOutput];
	if (created.includes("README.md")) {
		actions.push(openReadme);
	}

	const picked = await vscode.window.showInformationMessage(
		`${parts.join(", ")} in ${vscode.workspace.asRelativePath(targetDir)}. Next: go mod tidy && make run`,
		...actions,
	);
	if (picked === showOutput) {
		output.show(true);
	} else if (picked === openReadme) {
		const readme = vscode.Uri.joinPath(vscode.Uri.file(targetDir), "README.md");
		await vscode.window.showTextDocument(await vscode.workspace.openTextDocument(readme));
	}
}

async function reportFailure(
	err: unknown,
	output: vscode.OutputChannel,
): Promise<void> {
	if (err instanceof BinaryNotFoundError) {
		output.appendLine(err.message);

		const install = "Install with go";
		const setPath = "Set path…";
		const picked = await vscode.window.showErrorMessage(
			"The scaffold CLI was not found.",
			{
				modal: true,
				detail:
					"This extension drives the CLI so the templates have a single source of truth.\n\n" +
					`Install it with:\n  ${INSTALL_COMMAND}\n\n` +
					`If it is already installed somewhere unusual, set "${CONFIG_SECTION}.binaryPath".`,
			},
			install,
			setPath,
		);

		if (picked === install) {
			// A terminal, not a background process: `go install` can take a while
			// and the user should see it, including any Go toolchain error.
			const terminal = vscode.window.createTerminal("Install scaffold CLI");
			terminal.show();
			terminal.sendText(INSTALL_COMMAND);
		} else if (picked === setPath) {
			await vscode.commands.executeCommand(
				"workbench.action.openSettings",
				`${CONFIG_SECTION}.binaryPath`,
			);
		}
		return;
	}

	if (err instanceof OutdatedBinaryError) {
		output.appendLine(err.message);

		const update = "Update with go";
		const picked = await vscode.window.showErrorMessage(
			"The scaffold CLI found on this machine is too old.",
			{
				modal: true,
				detail:
					`Found ${err.command} reporting version ${err.version}.\n\n` +
					"That build ignores the folder you pick and writes next to its own " +
					"executable, so nothing would appear where you asked. Its version " +
					"number is not usable to detect this — it was hardcoded — so the " +
					"extension checks for the `--force` flag instead.\n\n" +
					`Update it with:\n  ${INSTALL_COMMAND}`,
			},
			update,
		);
		if (picked === update) {
			const terminal = vscode.window.createTerminal("Update scaffold CLI");
			terminal.show();
			terminal.sendText(INSTALL_COMMAND);
		}
		return;
	}

	const message = err instanceof Error ? err.message : String(err);
	if (err instanceof GenerationError) {
		output.appendLine(err.output);
	} else {
		output.appendLine(message);
	}

	const showOutput = "Show Output";
	const picked = await vscode.window.showErrorMessage(message, showOutput);
	if (picked === showOutput) {
		output.show(true);
	}
}

async function isDirectory(fsPath: string): Promise<boolean> {
	try {
		return (await fs.stat(fsPath)).isDirectory();
	} catch {
		return false;
	}
}
