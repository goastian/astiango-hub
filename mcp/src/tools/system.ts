import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";

const SYSTEM_TOOLS = {
  health_check: "astiango_hub_health_check",
  system_status: "astiango_hub_system_status",
  get_system_overview: "astiango_hub_get_system_overview",
};

export function configureSystemTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    SYSTEM_TOOLS.health_check,
    "Check if AstianGO Hub system is healthy and reachable",
    {},
    async () => {
      try {
        const isHealthy = await client.healthCheck();
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                healthy: isHealthy,
                status: isHealthy ? "System is healthy and reachable" : "System is not reachable",
                timestamp: new Date().toISOString(),
              }, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error checking system health: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SYSTEM_TOOLS.system_status,
    "Get comprehensive system status including nodes, recent tasks, and schedules",
    {},
    async () => {
      try {
        // Get system overview data
        const [nodesResponse, tasksResponse, schedulesResponse] = await Promise.all([
          client.getNodes({ page: 1, size: 50 }),
          client.getTasks({ page: 1, size: 20 }),
          client.getSchedules({ page: 1, size: 50 }),
        ]);

        const summary = {
          timestamp: new Date().toISOString(),
          nodes: {
            total: nodesResponse.total || 0,
            online: nodesResponse.data?.filter(node => node.status === 'online').length || 0,
            offline: nodesResponse.data?.filter(node => node.status === 'offline').length || 0,
          },
          tasks: {
            total_recent: tasksResponse.total || 0,
            running: tasksResponse.data?.filter(task => task.status === 'running').length || 0,
            pending: tasksResponse.data?.filter(task => task.status === 'pending').length || 0,
            assigned: tasksResponse.data?.filter(task => task.status === 'assigned').length || 0,
            finished: tasksResponse.data?.filter(task => task.status === 'finished').length || 0,
            error: tasksResponse.data?.filter(task => task.status === 'error').length || 0,
            cancelled: tasksResponse.data?.filter(task => task.status === 'cancelled').length || 0,
            abnormal: tasksResponse.data?.filter(task => task.status === 'abnormal').length || 0,
          },
          schedules: {
            total: schedulesResponse.total || 0,
            enabled: schedulesResponse.data?.filter(schedule => schedule.enabled).length || 0,
            disabled: schedulesResponse.data?.filter(schedule => !schedule.enabled).length || 0,
          },
          health: await client.healthCheck(),
        };

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(summary, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting system status: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SYSTEM_TOOLS.get_system_overview,
    "Get a high-level overview of the AstianGO Hub system including projects, spiders, nodes, and activity",
    {},
    async () => {
      try {
        // Get comprehensive system data
        const [projectsResponse, spidersResponse, nodesResponse, tasksResponse, schedulesResponse] = await Promise.all([
          client.getProjects({ page: 1, size: 100 }),
          client.getSpiders({ page: 1, size: 100 }),
          client.getNodes({ page: 1, size: 50 }),
          client.getTasks({ page: 1, size: 50 }),
          client.getSchedules({ page: 1, size: 100 }),
        ]);

        const overview = {
          timestamp: new Date().toISOString(),
          system_health: await client.healthCheck(),
          summary: {
            projects: projectsResponse.total || 0,
            spiders: spidersResponse.total || 0,
            nodes: nodesResponse.total || 0,
            recent_tasks: tasksResponse.total || 0,
            schedules: schedulesResponse.total || 0,
          },
          nodes_detail: {
            total: nodesResponse.total || 0,
            master_nodes: nodesResponse.data?.filter(node => node.is_master).length || 0,
            worker_nodes: nodesResponse.data?.filter(node => !node.is_master).length || 0,
            active: nodesResponse.data?.filter(node => node.active).length || 0,
            enabled: nodesResponse.data?.filter(node => node.enabled).length || 0,
          },
          tasks_detail: {
            by_status: {
              pending: tasksResponse.data?.filter(task => task.status === 'pending').length || 0,
              assigned: tasksResponse.data?.filter(task => task.status === 'assigned').length || 0,
              running: tasksResponse.data?.filter(task => task.status === 'running').length || 0,
              finished: tasksResponse.data?.filter(task => task.status === 'finished').length || 0,
              error: tasksResponse.data?.filter(task => task.status === 'error').length || 0,
              cancelled: tasksResponse.data?.filter(task => task.status === 'cancelled').length || 0,
              abnormal: tasksResponse.data?.filter(task => task.status === 'abnormal').length || 0,
            }
          },
          schedules_detail: {
            total: schedulesResponse.total || 0,
            enabled: schedulesResponse.data?.filter(schedule => schedule.enabled).length || 0,
            disabled: schedulesResponse.data?.filter(schedule => !schedule.enabled).length || 0,
          },
        };

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(overview, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting system overview: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { SYSTEM_TOOLS };
