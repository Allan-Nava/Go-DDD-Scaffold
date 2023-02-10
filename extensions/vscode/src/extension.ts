// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import * as _ from "lodash";
import * as mkdirp from "mkdirp";
import { newScaffold } from "./commands";
//
let homeDir 		= os.homedir();
let CONFIG_PATH : string;
//
// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {
	//
	// Use the console to output diagnostic information (console.log) and errors (console.error)
	// This line of code will only be executed once when your extension is activated
	console.log('Congratulations, your extension "go-ddd-scaffold" is now active!');
	//
	CONFIG_PATH = getConfigPath(CONFIG_NAME);
	/// https://github.com/Microsoft/vscode-extension-samples/blob/master/configuration-sample/src/extension.ts
	//console.log(ConfigurationTarget.Workspace);
	console.log(`CONFIG_PATH = ${CONFIG_PATH}`);
	//
	// The command has been defined in the package.json file
	// Now provide the implementation of the command with registerCommand
	// The commandId parameter must match the command field in package.json
	let disposable = vscode.commands.registerCommand('go-ddd-scaffold.helloWorld', () => {
		// The code you place here will be executed every time your command is executed
		// Display a message box to the user
		vscode.window.showInformationMessage('Hello World from go-ddd-scaffold!');
	});
	//
	context.subscriptions.push(disposable);
	//
	context.subscriptions.push(
		vscode.commands.registerCommand("extension.new-scaffold", newScaffold),
	);
}
//
// This method is called when your extension is deactivated
export function deactivate() {}
//
///
window.onDidChangeActiveTextEditor(function(event){
	console.log("onDidChangeActiveTextEditor "+event);
});
//
///
async function promptForTargetFiles(): Promise<string | undefined> {
	console.log("promptForTargetFiles()");
	const options: OpenDialogOptions = {
	  canSelectMany: false,
	  openLabel: "Select a file to upload",
	  canSelectFolders: false,
	  canSelectFiles: true,
	};
  
	return window.showOpenDialog(options).then(uri => {
	  if (_.isNil(uri) || _.isEmpty(uri)) {
		return undefined;
	  }
	  return uri[0].fsPath;
	});
}
///

//
function createDirectory(targetDirectory: string): Promise<void> {
	return new Promise((resolve, reject) => {
	  mkdirp(targetDirectory, { mode: '0777' }).then(made => {
		  if(made){
			  return reject(made);
		  }
		  resolve();
	  });
	});
}
///