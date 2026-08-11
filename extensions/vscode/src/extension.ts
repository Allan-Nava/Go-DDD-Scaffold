// Go DDD Scaffold — VS Code extension.
//
// The extension is a wrapper around the `scaffold` CLI: it does not carry its
// own copy of the templates. That is deliberate. The previous version kept a
// second set of templates under src/templates/, which drifted from the CLI's
// (one file was a verbatim duplicate of another) and was never read by any code
// path. With the CLI as the single implementation, that class of bug is gone.
import * as vscode from "vscode";

import { newScaffoldProject } from "./commands/init";

export function activate(context: vscode.ExtensionContext): void {
	// One channel for the whole session: the CLI's per-file output goes here, so
	// a generation can be inspected after the notification is gone.
	const output = vscode.window.createOutputChannel("Go DDD Scaffold");
	context.subscriptions.push(output);

	context.subscriptions.push(
		vscode.commands.registerCommand(
			"go-ddd-scaffold.init",
			(uri?: vscode.Uri) => newScaffoldProject(output, uri),
		),
	);
}

export function deactivate(): void {
	// Nothing to tear down: everything is registered in context.subscriptions.
}
