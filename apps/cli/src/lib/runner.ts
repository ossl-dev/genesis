import {
  collectPluginInstances,
  loadPlugins,
  buildPluginGraph,
  runApply as coreRunApply,
  runDiff as coreRunDiff,
  runValidate as coreRunValidate,
  runDetect as coreRunDetect,
  loadConfig,
  Logger,
  TaskRegistry,
  type DetectSummary,
  type ApplySummary,
  type ValidateSummary,
} from "@ossl/genesis-core";

export interface RunnerContext {
  cwd: string;
  configPath?: string;
}

function createLogger(): Logger {
  const level = process.env.GENESIS_DEBUG ? "debug" : "info";
  return new Logger({ level, prefix: "genesis" });
}

function printTable(
  rows: { plugin: string; status: string; details?: string }[]
): void {
  if (rows.length === 0) {
    return;
  }
  const headerPlugin = "Plugin";
  const headerStatus = "Status";
  const pluginWidth = Math.max(
    headerPlugin.length,
    ...rows.map((row) => row.plugin.length)
  );
  const statusWidth = Math.max(
    headerStatus.length,
    ...rows.map((row) => row.status.length)
  );
  const line = `${headerPlugin.padEnd(pluginWidth)}  ${headerStatus.padEnd(
    statusWidth
  )}  Details`;
  console.log(line);
  for (const row of rows) {
    const text = `${row.plugin.padEnd(pluginWidth)}  ${row.status.padEnd(
      statusWidth
    )}  ${row.details ?? ""}`;
    console.log(text);
  }
}

async function prepare(context: RunnerContext) {
  const logger = createLogger();
  logger.debug(`Loading config from ${context.cwd}`);
  const config = await loadConfig(context.cwd, context.configPath);
  const instances = collectPluginInstances(config);
  const nodes = await loadPlugins(instances);
  const graph = buildPluginGraph(nodes);
  return { logger, graph, env: { ...process.env, ...config.env } };
}

export async function runApply(context: RunnerContext): Promise<void> {
  const { logger, graph, env } = await prepare(context);
  const taskRegistry = new TaskRegistry(logger);

  logger.info(`Applying ${graph.length} plugins`);
  const summaries: ApplySummary[] = await coreRunApply(graph, {
    cwd: context.cwd,
    env,
    logger,
    taskRegistry,
  });
  const rows = summaries.map((summary) => ({
    plugin: summary.id,
    status: summary.ok ? "ok" : "error",
    details: summary.details,
  }));
  printTable(rows);
  if (summaries.some(summary => !summary.ok)) throw new Error("Environment apply failed");
}

export async function runDoctor(context: RunnerContext): Promise<void> {
  const { logger, graph, env } = await prepare(context);
  const taskRegistry = new TaskRegistry(logger);

  logger.info("Running diagnostics");
  const detectSummaries: DetectSummary[] = await coreRunDetect(graph, {
    cwd: context.cwd,
    env,
    logger,
    taskRegistry,
  });
  const validateSummaries: ValidateSummary[] = await coreRunValidate(graph, {
    cwd: context.cwd,
    env,
    logger,
    taskRegistry,
  });
  const rows = detectSummaries.map((detect) => {
    const validate = validateSummaries.find((entry) => entry.id === detect.id);
    const status =
      detect.status === "present" && validate?.ok
        ? "ok"
        : detect.status === "missing"
        ? "missing"
        : validate && !validate.ok
        ? "error"
        : "unknown";
    const details = validate?.message ?? detect.details;
    return {
      plugin: detect.id,
      status,
      details,
    };
  });
  printTable(rows);
  if (detectSummaries.some(summary => summary.status === "missing") || validateSummaries.some(summary => !summary.ok)) {
    throw new Error("Environment diagnostics failed");
  }
}

export async function runDiff(context: RunnerContext): Promise<void> {
  const { logger, graph, env } = await prepare(context);
  const taskRegistry = new TaskRegistry(logger);

  logger.info("Computing diff");
  const summaries: DetectSummary[] = await coreRunDiff(graph, {
    cwd: context.cwd,
    env,
    logger,
    taskRegistry,
  });
  const rows = summaries.map((summary) => ({
    plugin: summary.id,
    status: summary.status,
    details: summary.details,
  }));
  printTable(rows);
}

export async function runValidate(context: RunnerContext): Promise<void> {
  const { logger, graph, env } = await prepare(context);
  const taskRegistry = new TaskRegistry(logger);

  logger.info("Validating environment");
  const summaries: ValidateSummary[] = await coreRunValidate(graph, {
    cwd: context.cwd,
    env,
    logger,
    taskRegistry,
  });
  const rows = summaries.map((summary) => ({
    plugin: summary.id,
    status: summary.ok ? "ok" : "error",
    details: summary.message,
  }));
  printTable(rows);
  if (summaries.some(summary => !summary.ok)) throw new Error("Environment validation failed");
}
