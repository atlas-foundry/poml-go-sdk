#!/usr/bin/env node
/**
 * Best-effort TypeScript/Node benchmarks against the published poml package.
 * Tries to require common package names and falls back to the CLI if present.
 * Emits a JSON report to the provided path (argv[2]).
 */

const fs = require("fs");
const { performance } = require("perf_hooks");
const { spawnSync } = require("child_process");
const path = require("path");

const outputPath = process.argv[2] || "ts_bench.json";

const benchDoc = `<poml>
  <meta><id>bench.demo</id><version>1.0</version><owner>tester</owner></meta>
  <role>Bench role</role>
  <task>Do a thing</task>
  <system-msg>System hello</system-msg>
  <human-msg>Hello</human-msg>
</poml>`;

function writeResult(payload) {
  fs.writeFileSync(outputPath, JSON.stringify(payload, null, 2));
}

function tryRequire() {
  const candidates = ["poml", "@microsoft/poml", "@microsoft/poml-sdk"];
  for (const name of candidates) {
    try {
      const mod = require(name);
      return { mod, name };
    } catch (err) {
      continue;
    }
  }
  return null;
}

function measure(name, fn, iterations = 300) {
  const t0 = performance.now();
  let err;
  for (let i = 0; i < iterations; i++) {
    try {
      fn();
    } catch (e) {
      err = e;
      break;
    }
  }
  const nsPerOp = ((performance.now() - t0) * 1e6) / iterations;
  const entry = { name, ns_per_op: nsPerOp };
  if (err) {
    entry.error = String(err);
  }
  return entry;
}

function benchModule(mod) {
  const results = [];
  const candidate = mod.poml || mod.default || mod;
  if (typeof candidate !== "function") {
    return [{ name: "module_import", error: "No callable entry point (expected poml())" }];
  }
  const fn = candidate;
  results.push(measure("parse_message_dict", () => fn(benchDoc, { format: "message_dict" })));
  results.push(measure("parse_dict", () => fn(benchDoc, { format: "dict" })));
  results.push(measure("parse_openai_chat", () => fn(benchDoc, { format: "openai_chat" })));
  results.push(measure("parse_langchain", () => fn(benchDoc, { format: "langchain" })));
  return results;
}

function benchCli() {
  const results = [];
  const tmpPath = path.join(process.cwd(), "ts_bench_tmp.poml");
  fs.writeFileSync(tmpPath, benchDoc, "utf8");
  const cli = process.env.POML_CLI || "poml";
  const cmd = spawnSync(cli, ["-f", tmpPath, "-o", "-", "--format", "dict"], {
    encoding: "utf8",
  });
  if (cmd.error) {
    results.push({ name: "cli_dict", error: String(cmd.error) });
    return results;
  }
  if (cmd.status !== 0) {
    results.push({ name: "cli_dict", error: cmd.stderr || `exit ${cmd.status}` });
    return results;
  }
  // Measure multiple runs quickly using the same CLI call.
  const iterations = 50;
  const t0 = performance.now();
  for (let i = 0; i < iterations; i++) {
    spawnSync(cli, ["-f", tmpPath, "-o", "-", "--format", "dict"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
  }
  const nsPerOp = ((performance.now() - t0) * 1e6) / iterations;
  results.push({ name: "cli_dict", ns_per_op: nsPerOp });
  return results;
}

function main() {
  const found = tryRequire();
  let payload = { package: null, benchmarks: [] };
  if (found) {
    payload.package = found.name;
    payload.benchmarks.push(...benchModule(found.mod));
  } else {
    payload.package = "not-found";
    payload.benchmarks.push({ name: "module_import", error: "poml package not found in node_modules" });
  }
  // Try CLI fallback regardless.
  payload.benchmarks.push(...benchCli());
  writeResult(payload);
}

try {
  main();
} catch (err) {
  writeResult({ error: String(err), stack: err && err.stack });
}
