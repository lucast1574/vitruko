import { useCallback, useEffect, useState } from "react";
import { api } from "./client";
export function useData(path) {
  const [data, setData] = useState(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const reload = useCallback(async () => {
    setError("");
    try {
      const v = await api(path);
      setData(v);
      return v;
    } catch (e) {
      if (e.status === 401 && path !== "/console/session")
        location.assign("/login");
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [path]);
  useEffect(() => {
    let active = true;
    api(path)
      .then((v) => {
        if (active) setData(v);
      })
      .catch((e) => {
        if (active) {
          if (e.status === 401) location.assign("/login");
          setError(e.message);
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [path]);
  return { data, error, loading, reload, setData };
}
