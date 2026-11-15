import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./client.js";
export function useData(path) {
  const [data, setData] = useState(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const sequence = useRef(0), controller = useRef(null);
  const reload = useCallback(async () => {
    const request = ++sequence.current;
    controller.current?.abort();
    const current = new AbortController();
    controller.current = current;
    setLoading(true);
    setError("");
    try {
      const value = await api(path, "GET", undefined, {}, current.signal);
      if (request !== sequence.current) return;
      setData(value);
      return value;
    } catch (e) {
      if (request === sequence.current && e.name !== "AbortError") {
        if (e.status === 401) {
          const connect = new URLSearchParams(location.search).get("connect");
          location.assign("/login" + (["codex", "claude"].includes(connect) ? "?connect=" + connect : ""));
        }
        setError(e.message);
        // Callers must not announce success when refreshing the saved data fails.
        throw e;
      }
    } finally {
      if (request === sequence.current) setLoading(false);
    }
  }, [path]);
  useEffect(() => {
    setData(null);
    reload().catch(() => {});
    return () => {
      ++sequence.current;
      controller.current?.abort();
    };
  }, [reload]);
  return { data, error, loading, reload, setData };
}
