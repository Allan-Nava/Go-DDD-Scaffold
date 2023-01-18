import * as _ from "lodash";
import * as changeCase from "change-case";
import * as mkdirp from "mkdirp";
//
import {
    InputBoxOptions,
    OpenDialogOptions,
    Uri,
    window,
    workspace,
  } from "vscode";
import { existsSync, lstatSync } from "fs";
//
export const newScaffold = async (uri: Uri) => {
  const scaffoldName = await promptForScaffoldName();
  if (_.isNil(scaffoldName) || scaffoldName.trim() === "") {
    window.showErrorMessage("The scaffold name must not be empty");
    return;
  }
  //
  let targetDirectory;
  if (_.isNil(_.get(uri, "fsPath")) || !lstatSync(uri.fsPath).isDirectory()) {
    targetDirectory = await promptForTargetDirectory();
    if (_.isNil(targetDirectory)) {
      window.showErrorMessage("Please select a valid directory");
      return;
    }
  } else {
    targetDirectory = uri.fsPath;
  }
  //
  try {
    await generateScaffoldCode(scaffoldName, targetDirectory,);
    window.showInformationMessage(
      `Successfully Generated ${scaffoldName} Scaffold`
    );
  } catch (error) {
    window.showErrorMessage(
      `Error:
        ${error instanceof Error ? error.message : JSON.stringify(error)}`
    );
  }
};
//
function createScaffoldTemplate(
    scaffoldName: string,
    targetDirectory: string,
  ){
    const snakeCaseScaffoldName = changeCase.snakeCase(scaffoldName);
    console.log("snakeCaseScaffoldName", snakeCaseScaffoldName, targetDirectory);

}
//
function promptForScaffoldName(): Thenable<string | undefined> {
  const blocNamePromptOptions: InputBoxOptions = {
    prompt: "Scaffold Name Name",
    placeHolder: "GO DDD",
  };
  return window.showInputBox(blocNamePromptOptions);
}
//

async function promptForTargetDirectory(): Promise<string | undefined> {
  const options: OpenDialogOptions = {
    canSelectMany: false,
    openLabel: "Select a folder to create the scaffold in",
    canSelectFolders: true,
  };

  return window.showOpenDialog(options).then((uri) => {
    if (_.isNil(uri) || _.isEmpty(uri)) {
      return undefined;
    }
    return uri[0].fsPath;
  });
}
//
async function generateScaffoldCode(
  scaffoldName: string,
  targetDirectory: string,
) {
  const shouldCreateDirectory = workspace
    .getConfiguration("bloc")
    .get<boolean>("newBlocTemplate.createDirectory");
  const scaffoldDirectoryPath = shouldCreateDirectory
    ? `${targetDirectory}/bloc`
    : targetDirectory;
  if (!existsSync(scaffoldDirectoryPath)) {
    await createDirectory(scaffoldDirectoryPath);
  }
  //
  await Promise.all([
    createScaffoldTemplate(scaffoldName, scaffoldDirectoryPath),
  ]);
}
//
function createDirectory(targetDirectory: string): Promise<void>  {
  return new Promise((resolve, reject) => {
    mkdirp(targetDirectory, (error) => {
      if (error) {
        return reject(error);
      }
      resolve();
    });
  });
}