import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";

export function configurePrompts(server: McpServer) {
  server.prompt(
    "spider-analysis",
    "Analyze spider performance and provide insights",
    {
      spider_id: z.string().describe("ID of the spider to analyze"),
      time_range: z.string().optional().describe("Time range for analysis (e.g., '7d', '30d', '90d')"),
    },
    async (args) => {
      const spiderId = args.spider_id as string;
      const timeRange = (args.time_range as string) || "7d";
      
      return {
        description: `Spider Performance Analysis for Spider ${spiderId}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `Please analyze the performance of spider "${spiderId}" over the last ${timeRange}. 

I would like to understand:
1. Task success/failure rates
2. Average execution time
3. Data collection patterns
4. Any errors or issues
5. Recommendations for optimization

Please use the available AstianGO Hub tools to gather task data, logs, and results for this analysis.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "task-debugging",
    "Help debug a failed task",
    {
      task_id: z.string().describe("ID of the failed task to debug"),
    },
    async (args) => {
      const taskId = args.task_id as string;
      
      return {
        description: `Task Debugging Assistant for Task ${taskId}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `I have a failed task with ID "${taskId}" that needs debugging. 

Please help me:
1. Get the task details and status
2. Retrieve and analyze the task logs
3. Identify the root cause of the failure
4. Suggest specific fixes or improvements
5. Provide guidance on preventing similar issues

Use the AstianGO Hub tools to gather all relevant information about this task.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "spider-setup",
    "Guide for setting up a new spider",
    {
      spider_name: z.string().describe("Name for the new spider"),
      target_website: z.string().optional().describe("Target website or data source"),
      spider_type: z.string().optional().describe("Type of spider (scrapy, selenium, custom, etc.)"),
    },
    async (args) => {
      const spiderName = args.spider_name as string;
      const targetWebsite = args.target_website as string;
      const spiderType = (args.spider_type as string) || "scrapy";
      
      return {
        description: `Spider Setup Guide for "${spiderName}"`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `I want to create a new ${spiderType} spider named "${spiderName}"${targetWebsite ? ` to scrape ${targetWebsite}` : ""}.

Please help me:
1. Create the spider configuration
2. Set up the basic file structure
3. Configure appropriate settings
4. Create a simple test to verify the setup
5. Set up a schedule if needed

Guide me through the process step by step using the available AstianGO Hub tools.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "system-monitoring",
    "Monitor AstianGO Hub system health and performance",
    {
      focus_area: z.string().optional().describe("Specific area to focus on (nodes, tasks, storage, overall)"),
    },
    async (args) => {
      const focusArea = (args.focus_area as string) || "overall";
      
      return {
        description: `AstianGO Hub System Monitoring - ${focusArea}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `Please perform a system health check focusing on ${focusArea}.

I need you to:
1. Check node status and availability
2. Review recent task performance
3. Identify any system bottlenecks
4. Check for error patterns
5. Provide recommendations for optimization

Use the AstianGO Hub monitoring tools to gather comprehensive system information.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "ai-autoprobe-setup",
    "Guide for setting up AI-powered web scraping with AutoProbe",
    {
      target_url: z.string().describe("Target URL to scrape"),
      data_description: z.string().describe("Description of the data you want to extract"),
      autoprobe_name: z.string().optional().describe("Name for the AutoProbe configuration"),
    },
    async (args) => {
      const targetUrl = args.target_url as string;
      const dataDescription = args.data_description as string;
      const autoprobeName = (args.autoprobe_name as string) || `AutoProbe for ${new URL(targetUrl).hostname}`;
      
      return {
        description: `AI AutoProbe Setup Guide for ${targetUrl}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `I want to set up an AI-powered AutoProbe to scrape data from "${targetUrl}".

Target URL: ${targetUrl}
Data to extract: ${dataDescription}
AutoProbe name: ${autoprobeName}

Please help me:
1. Check if there are existing LLM providers configured
2. Create a new AutoProbe V2 configuration with the specified parameters
3. Run a preview to see what data would be extracted
4. Test the AutoProbe with a sample run
5. Provide guidance on optimizing the extraction pattern

Use the AI and AutoProbe tools to guide me through this process.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "llm-provider-setup",
    "Configure LLM providers for AI features",
    {
      provider_type: z.string().describe("Type of LLM provider (openai, azure-openai, anthropic, gemini)"),
      provider_name: z.string().optional().describe("Display name for the provider"),
    },
    async (args) => {
      const providerType = args.provider_type as string;
      const providerName = (args.provider_name as string) || `${providerType.charAt(0).toUpperCase() + providerType.slice(1)} Provider`;
      
      return {
        description: `LLM Provider Setup Guide for ${providerType}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `I want to set up a ${providerType} LLM provider named "${providerName}" for AstianGO Hub's AI features.

Please help me:
1. List existing LLM providers to avoid conflicts
2. Guide me through the configuration process for ${providerType}
3. Explain what API credentials and settings are needed
4. Test the provider connection
5. Show me how to use this provider with AutoProbe and other AI features

Use the LLM provider tools to assist with the setup.`
            }
          }
        ]
      };
    }
  );

  server.prompt(
    "ai-data-extraction",
    "Optimize AI-powered data extraction",
    {
      autoprobe_id: z.string().describe("ID of the AutoProbe to optimize"),
      extraction_issues: z.string().optional().describe("Specific issues with current extraction"),
    },
    async (args) => {
      const autoprobeId = args.autoprobe_id as string;
      const extractionIssues = args.extraction_issues as string;
      
      return {
        description: `AI Data Extraction Optimization for AutoProbe ${autoprobeId}`,
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `I need help optimizing the AI data extraction for AutoProbe "${autoprobeId}".

${extractionIssues ? `Current issues: ${extractionIssues}` : ""}

Please help me:
1. Review the current AutoProbe configuration
2. Check the extraction pattern and recent results
3. Analyze any failed or incomplete extractions
4. Suggest improvements to the query or settings
5. Test the optimized configuration

Use the AutoProbe tools to analyze and improve the extraction process.`
            }
          }
        ]
      };
    }
  );
}
