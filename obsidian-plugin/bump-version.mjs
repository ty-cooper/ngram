import { readFileSync, writeFileSync } from "fs";

function bump(file) {
  const data = JSON.parse(readFileSync(file, "utf8"));
  const parts = data.version.split(".").map(Number);
  parts[2]++;
  data.version = parts.join(".");
  writeFileSync(file, JSON.stringify(data, null, 2) + "\n");
  return data.version;
}

const v = bump("package.json");
bump("manifest.json");
console.log(`bumped to ${v}`);
