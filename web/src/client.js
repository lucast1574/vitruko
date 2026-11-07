export async function api(path, method = "GET", body, headers = {}, signal) {
  const response = await fetch(path, {
    method,
    credentials: "same-origin",
    headers: {
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal,
  });
  let data;
  try {
    data = response.status === 204 ? null : await response.json();
  } catch {
    const error = new Error("El servidor no devolvió una respuesta válida. Intenta de nuevo.");
    error.status = response.status;
    throw error;
  }
  if (!response.ok) {
    const error = new Error(data?.error || "No se pudo completar la acción.");
    error.status = response.status;
    error.code = data?.code;
    throw error;
  }
  return data;
}
const localDevelopment = import.meta.env?.DEV && typeof location !== "undefined";
export const appURL = localDevelopment ? "" : "https://app-vitruko.nexode.app";
export const siteURL = localDevelopment ? "" : "https://vitruko.nexode.app";
export const number = (value) =>
  new Intl.NumberFormat("es-PE").format(value || 0);
export const date = (value) =>
  new Date(value).toLocaleDateString("es-PE", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
