import Ajv from "ajv";
import { glob, readFile } from "node:fs/promises";

const schema = JSON.parse(await readFile("schema/envbuckets.schema.json", "utf8"));
const product = await readFile("docs-eb/PRODUCT.md", "utf8");
const examples = [...product.matchAll(/```json\s*\n([\s\S]*?)\n```/g)];
if (examples.length === 0) throw new Error("PRODUCT.md has no JSON config examples");

const validate = new Ajv({ allErrors: true }).compile(schema);
for (let i = 0; i < examples.length; i++) {
  const value = JSON.parse(examples[i][1]);
  if (!validate(value)) {
    const details = validate.errors
      .map((error) => `${error.instancePath || "/"} ${error.message}`)
      .join("; ");
    throw new Error(`PRODUCT.md JSON example ${i + 1} fails schema: ${details}`);
  }
}
console.log(`Validated ${examples.length} PRODUCT.md config examples against draft-07 schema.`);

let invalidCount = 0;
for await (const path of glob("schema/testdata/invalid/*.json")) {
  const value = JSON.parse(await readFile(path, "utf8"));
  if (validate(value)) throw new Error(`${path} unexpectedly passes the schema`);
  invalidCount++;
}
if (invalidCount === 0) throw new Error("No invalid schema fixtures found");
console.log(`Rejected ${invalidCount} invalid config fixtures against draft-07 schema.`);
