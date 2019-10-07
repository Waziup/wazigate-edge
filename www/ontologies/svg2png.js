
const { execSync } = require('child_process');
const fs = require('fs');

var files = fs.readdirSync('.');
files = files.filter((name) => name.endsWith(".svg"))
if (files.length == 0) {
    console.error('readdir: found no svg files in current directory');
    return;
}

var inkscape = "inkscape";
try {
    execSync(`${inkscape} --version`, {stdio: 'ignore'});
} catch (err) {
    inkscape = '"C:\\Program Files\\Inkscape\\inkscape.exe"'
    try {
        execSync(`${inkscape} --version`, {stdio: 'ignore'});
    } catch {
        throw err;
    }
}


for (var file of files) {

    var outFile = file.slice(0, -4)+".png";
    console.log(`${file} > ${outFile}`);

    execSync(`${inkscape} --without-gui --file="${file}" --export-png="${outFile}" --export-width 256 --export-height 256`);
}


