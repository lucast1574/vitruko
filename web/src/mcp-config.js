export function installPrompt(client, connection) {
  const target = client === "codex" ? "Codex" : "Claude Code";
  const configuration =
    client === "codex"
      ? `[mcp_servers.vitruko]\nurl = ${JSON.stringify(connection.url)}\n[mcp_servers.vitruko.http_headers]\nAuthorization = ${JSON.stringify("Bearer " + connection.key)}`
      : JSON.stringify(
          {
            mcpServers: {
              vitruko: {
                type: "http",
                url: connection.url,
                headers: { Authorization: "Bearer " + connection.key },
              },
            },
          },
          null,
          2,
        );
  return `Conecta mi cuenta de Vitruko a ${target} usando esta configuración MCP HTTP. Instálala únicamente en la configuración personal de ${target} (TOML en ~/.codex/config.toml para Codex; configuración de usuario para Claude Code). Conserva los demás servidores y actualiza solo vitruko si ya existe. La API key ya está incluida: no me pidas crear ni copiar otra. No guardes la credencial en el proyecto, Git, documentación ni memoria; no la imprimas ni la repitas en la respuesta. Mantén las confirmaciones del asistente. Recarga el MCP o indícame cómo reabrir el asistente. Ejecuta vitruko_me para verificar la cuenta y vitruko_documentation para leer el contrato. No hagas consultas de datos ni operaciones que consuman cuota durante la verificación.\n\n${configuration}`;
}
