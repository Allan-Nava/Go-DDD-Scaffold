// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import * as _ from "lodash";
import * as mkdirp from "mkdirp";
import * as changeCase from "change-case";
import { existsSync, lstatSync, writeFile } from "fs";

//import * from './templates';
import { newScaffold } from "./commands";
//
//
// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {
	//
	// Use the console to output diagnostic information (console.log) and errors (console.error)
	// This line of code will only be executed once when your extension is activated
	console.log('Congratulations, your extension "go-ddd-scaffold" is now active!');
	//
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
	/*context.subscriptions.push(
		vscode.commands.registerCommand("extension.new-scaffold", newScaffold),
	);*/
	// The command has been defined in the package.json file
	// Now provide the implementation of the command with registerCommand
	// The commandId parameter must match the command field in package.json
	let new_scaffold = vscode.commands.registerCommand('extension.new-scaffold', async (uri: vscode.Uri) => {
		// The code you place here will be executed every time your command is executed
		// Display a message box to the user
		//window.showInformationMessage('Hello World from Vanilla BLoC!');
		console.log("extension.new-scaffold starts");
		const scaffoldName = await promptForBlocName();
		if (_.isNil(scaffoldName) || scaffoldName.trim() === "") {
			vscode.window.showErrorMessage("The bloc name must not be empty");
			return;
		}
		console.log("scaffoldName", scaffoldName);
		let targetDirectory;
		if (_.isNil(_.get(uri, "fsPath")) || !lstatSync(uri.fsPath).isDirectory()) {
			targetDirectory = await promptForTargetDirectory();
			if (_.isNil(targetDirectory)) {
				vscode.window.showErrorMessage("Please select a valid directory");
				return;
			}
		} else {
			targetDirectory = uri.fsPath;
		}
		console.log(`targetDirectory ${targetDirectory}`);
		try {
			await generateScaffoldCode( scaffoldName, targetDirectory );
			vscode.window.showInformationMessage(
			  `Successfully Generated ${scaffoldName}`
			);
		} catch (error) {
			console.log("error", error);
			vscode.window.showErrorMessage(
			  `Error:
			  ${error instanceof Error ? error.message : JSON.stringify(error)}`
			);
		}
	});
	///
	context.subscriptions.push(new_scaffold);
}
//
// This method is called when your extension is deactivated
export function deactivate() {}
//
///
vscode.window.onDidChangeActiveTextEditor(function(event){
	console.log("onDidChangeActiveTextEditor "+event);
});
//
///
async function promptForTargetFiles(): Promise<string | undefined> {
	console.log("promptForTargetFiles()");
	const options: vscode.OpenDialogOptions = {
	  canSelectMany: false,
	  openLabel: "Select a file to upload",
	  canSelectFolders: false,
	  canSelectFiles: true,
	};
  
	return vscode.window.showOpenDialog(options).then(uri => {
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

function promptForBlocName(): Thenable<string | undefined> {
	const blocNamePromptOptions: vscode.InputBoxOptions = {
	  prompt: "Vanilla Bloc Name",
	  placeHolder: "counter"
	};
	return vscode.window.showInputBox(blocNamePromptOptions);
}

async function promptForTargetDirectory(): Promise<string | undefined> {
	console.log("promptForTargetDirectory()");
	const options: vscode.OpenDialogOptions = {
	  canSelectMany: false,
	  openLabel: "Select a folder to create the bloc in",
	  canSelectFolders: true
	};
  
	return vscode.window.showOpenDialog(options).then(uri => {
	  if (_.isNil(uri) || _.isEmpty(uri)) {
		return undefined;
	  }
	  return uri[0].fsPath;
	});
}



async function generateScaffoldCode(
	scaffoldName: string,
	targetDirectory: string,
  ) {
	console.log(`generateBlocCode targetDirectory ${targetDirectory}`);
	const scaffoldDirectoryPath = `${targetDirectory}/bloc`;
	console.log(`scaffoldDirectoryPath ${scaffoldDirectoryPath}`);
	if (!existsSync(scaffoldDirectoryPath)) {
	  await createDirectoryV2(scaffoldDirectoryPath);
	}
	///
	await Promise.all([
	  createScaffoldTemplate(scaffoldName, targetDirectory,),    
	]);
}

function createDirectoryV2(targetDirectory: string): Promise<void> {
	console.log(`createDirectoryV2 ${targetDirectory}`);
	return new Promise((resolve, reject) => {
		mkdirp(targetDirectory, { mode: '0777' })
		resolve();
	});
}


function createScaffoldTemplate(scaffoldName: string, targetDirectory: string, ) {
	const snakeCaseScaffoldName = changeCase.snakeCase(scaffoldName.toLowerCase());
	const targetPath 			= `${targetDirectory}/bloc/${snakeCaseScaffoldName}_bloc.dart`;
	console.log(`targetPath ${targetPath}`);
	if (existsSync(targetPath)) {
	  throw Error(`${snakeCaseScaffoldName}_bloc.dart already exists`);
	}
	/*return new Promise(async (resolve, reject) => {
	  writeFile(targetPath, getScaffoldTemplate(scaffoldName), "utf8", error => {
		if (error) {
		  reject(error);
		  return;
		}
		resolve();
	  });
	});*/
}