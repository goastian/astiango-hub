#!/usr/bin/env node

/**
 * Simple build validation test for AstianGO Hub MCP Server
 */

import { AstianGoHubClient } from "./dist/client.js";
import { access } from "node:fs/promises";

console.log("🧪 Testing AstianGO Hub MCP Server build...\n");

// Test that we can instantiate the client
try {
  const client = new AstianGoHubClient("http://localhost:8080", "test-token");
  console.log("✅ AstianGoHubClient class - OK");
} catch (error) {
  console.log("❌ AstianGoHubClient class - FAILED");
  console.log("   Error:", error.message);
  process.exit(1);
}

// TypeScript already validates the module graph during build. Avoid importing
// the CLI entrypoint here because a CLI import intentionally parses argv.
try {
  await access(new URL("./dist/index.js", import.meta.url));
  console.log("✅ Main entry point - OK");
} catch (error) {
  console.log("❌ Main entry point - FAILED");
  console.log("   Error:", error.message);
  process.exit(1);
}

// Test tools module
try {
  const toolsModule = await import("./dist/tools.js");
  if (typeof toolsModule.configureAllTools === 'function') {
    console.log("✅ Tools configuration - OK");
  } else {
    console.log("❌ Tools configuration - FAILED (configureAllTools not a function)");
  }
} catch (error) {
  console.log("❌ Tools configuration - FAILED");
  console.log("   Error:", error.message);
}

console.log("\n🎉 Build validation completed successfully!");
console.log("\n📋 Ready to use:");
console.log("   npm start <astiango_hub_url> [api_token]");
console.log("   Example: npm start http://localhost:8080 your-token");
console.log("\n   Or use the binary directly:");
console.log("   ./dist/index.js http://localhost:8080 your-token");
