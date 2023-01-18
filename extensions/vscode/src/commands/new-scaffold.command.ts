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
import { existsSync, lstatSync, writeFile } from "fs";
//
export const newScaffold = async (uri: Uri) => {

}

function createScaffoldTemplate(
    scaffoldName: string,
    targetDirectory: string,
    type: BlocType
  ){
    const snakeCaseScaffoldName = changeCase.snakeCase(scaffoldName);

}