
const fs = require('fs');
const cheerio = require('cheerio')

var files = fs.readdirSync('.');
files = files.filter((name) => name.endsWith(".svg"))
if (files.length == 0) {
    console.error('readdir: found no svg files in current directory');
    return;
}

var map = `<svg aria-hidden="true" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
  <!-- © 2019 WAZIUP -->
  <!-- AUTO GENERATED FILE - DO NOT MODIFY! -->
  <!-- use \`node svg2map\` -->
<defs>
`;

for (var file of files) {
	var data = fs.readFileSync(file, 'utf8');
    var $ = cheerio.load(data, {xmlMode: true});
	var viewBox = $("svg").attr("viewBox");
	map += `<symbol id="${file.substr(0,file.length-4)}" viewBox="${viewBox}">\n`
	map += $('svg').html();
	map += `</symbol>\n`;
}

map += `
</defs>
</svg>
`;

fs.writeFileSync('../ontologies.svg', map);


