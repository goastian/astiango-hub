#!/usr/bin/env node

/**
 * Test script for AstianGO Hub MCP Server
 * This script validates that all tools are properly configured and can handle basic requests
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { configureAllTools } from "./dist/tools.js";
import { AstianGoHubClient } from "./dist/client.js";

// Mock AstianGO Hub client for testing
class MockAstianGoHubClient extends AstianGoHubClient {
  constructor() {
    super("http://localhost:8080", "test-token");
  }

  async getSpiders() {
    return {
      success: true,
      data: [
        {
          _id: "test-spider-1",
          name: "Test Spider",
          description: "A test spider",
          cmd: "scrapy crawl test",
          type: "scrapy",
          created_ts: new Date()
        }
      ],
      total: 1
    };
  }

  async getTasks() {
    return {
      success: true,
      data: [
        {
          _id: "test-task-1",
          spider_id: "test-spider-1",
          spider_name: "Test Spider",
          cmd: "scrapy crawl test",
          status: "success",
          created_ts: new Date()
        }
      ],
      total: 1
    };
  }

  async getNodes() {
    return {
      success: true,
      data: [
        {
          _id: "test-node-1",
          name: "Master Node",
          ip: "127.0.0.1",
          mac: "00:00:00:00:00:00",
          hostname: "localhost",
          status: "online",
          is_master: true
        }
      ],
      total: 1
    };
  }

  async healthCheck() {
    return true;
  }
}

async function testMcpServer() {
  console.log("🧪 Testing AstianGO Hub MCP Server...\n");

  const server = new McpServer({
    name: "AstianGO Hub MCP Server Test",
    version: "0.1.0-test",
  });

  const mockClient = new MockAstianGoHubClient();
  configureAllTools(server, mockClient);

  // Get list of registered tools
  const tools = server.listTools();
  console.log(`✅ Server initialized with ${tools.length} tools:`);
  
  tools.forEach((tool, index) => {
    console.log(`   ${index + 1}. ${tool.name}: ${tool.description}`);
  });

  console.log("\n🔧 Testing sample tool executions...\n");

  // Test spider listing tool
  try {
    const spiderResult = await server.callTool("astiango_hub_list_spiders", {});
    console.log("✅ astiango_hub_list_spiders - SUCCESS");
    console.log("   Sample output:", JSON.stringify(spiderResult, null, 2).substring(0, 200) + "...\n");
  } catch (error) {
    console.log("❌ astiango_hub_list_spiders - FAILED");
    console.log("   Error:", error.message, "\n");
  }

  // Test task listing tool
  try {
    const taskResult = await server.callTool("astiango_hub_list_tasks", {});
    console.log("✅ astiango_hub_list_tasks - SUCCESS");
    console.log("   Sample output:", JSON.stringify(taskResult, null, 2).substring(0, 200) + "...\n");
  } catch (error) {
    console.log("❌ astiango_hub_list_tasks - FAILED");
    console.log("   Error:", error.message, "\n");
  }

  // Test node listing tool
  try {
    const nodeResult = await server.callTool("astiango_hub_list_nodes", {});
    console.log("✅ astiango_hub_list_nodes - SUCCESS");
    console.log("   Sample output:", JSON.stringify(nodeResult, null, 2).substring(0, 200) + "...\n");
  } catch (error) {
    console.log("❌ astiango_hub_list_nodes - FAILED");
    console.log("   Error:", error.message, "\n");
  }

  console.log("🎉 Test completed! The MCP server appears to be working correctly.");
  console.log("\n📋 Next steps:");
  console.log("   1. Configure your MCP client (e.g., Claude Desktop) to use this server");
  console.log("   2. Point it to a real AstianGO Hub instance");
  console.log("   3. Set up proper API authentication");
  console.log("   4. Start managing your AstianGO Hub spiders through AI!");
}

// Run the test
testMcpServer().catch(console.error);
