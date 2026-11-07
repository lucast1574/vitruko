import React, { useState, useEffect } from "react";
import { ArrowRight, ArrowLeft } from "lucide-react";
import { Logo, Button, Notice } from "./ui";
import { api, siteURL } from "./client";
import { authDestination } from "./auth-flow.js";
export default function Auth() {
  const mode = location.pathname.slice(1);
  const plan = new URLSearchParams(location.search).get("plan");
  const planQuery = ["launch", "scale"].includes(plan) ? "?plan=" + plan : "";
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [message, setMessage] = useState("");
  const resetToken = new URLSearchParams(location.search).get("token") || "";
  const [google, setGoogle] = useState(null);
  useEffect(() => {
    if (mode === "login" || mode === "signup")
      api("/console/auth/providers")
        .then((v) => setGoogle(v.google))
        .catch(() => {});
  }, [mode]);
  async function withGoogle() {
    setBusy(true);
    setError("");
    setMessage("Elige tu cuenta en la ventana de Google.");
    try {
      const { googleIdentity } = await import("./google-client");
      const idToken = await googleIdentity(google);
      setMessage("Verificando tu acceso a Vitruko…");
      await api("/console/google", "POST", { idToken });
      location.assign(authDestination(mode, location.search));
    } catch (e) {
      const messages = {
        "auth/popup-closed-by-user":
          "Cerraste la ventana de Google. Puedes intentarlo de nuevo.",
        "auth/popup-blocked":
          "Permite la ventana emergente para acceder con Google.",
        "auth/network-request-failed":
          "No se pudo conectar con Google. Revisa tu conexión.",
        "auth/cancelled-popup-request": "Ya hay una ventana de Google abierta.",
      };
      setError(
        messages[e.code] ||
          (e.code?.startsWith("auth/")
            ? "No se pudo completar el acceso con Google. Intenta de nuevo."
            : e.message),
      );
    } finally {
      setBusy(false);
      setMessage("");
    }
  }
  const title =
    {
      login: "Tus datos te esperan.",
      signup: "Tu próximo punto.",
      forgot: "Recupera tu acceso.",
      reset: "Una nueva contraseña.",
    }[mode] || "Accede a Vitruko";
  async function submit(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setMessage("");
    const f = Object.fromEntries(new FormData(e.currentTarget));
    try {
      if (mode === "signup") {
        await api("/console/signup", "POST", {
          name: f.name,
          email: f.email,
          password: f.password,
        });
        location.assign(authDestination(mode, location.search));
      } else if (mode === "login") {
        await api("/console/login", "POST", {
          email: f.email,
          password: f.password,
        });
        location.assign(authDestination(mode, location.search));
      } else if (mode === "forgot") {
        const v = await api("/console/forgot", "POST", { email: f.email });
        setMessage(v.message);
      } else {
        await api("/console/reset", "POST", {
          token: resetToken,
          password: f.password,
        });
        history.replaceState(null, "", "/reset");
        setMessage("Contraseña actualizada. Ya puedes iniciar sesión.");
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="auth-page">
      <div className="auth-brand">
        <Logo />
        <a href={siteURL || "/"}>
          <ArrowLeft size={16} />
          Volver a Vitruko
        </a>
      </div>
      <main className="auth-main">
        <div className="auth-form-wrap">
          <h1>{title}</h1>
          <p>
            {mode === "signup"
              ? "1.000 peticiones al mes para empezar. Sin tarjeta."
              : mode === "forgot"
                ? "Te enviaremos un enlace para definir una contraseña nueva."
                : "Tus claves, tu consumo y tu próxima integración."}
          </p>
          <Notice error>{error}</Notice>
          <Notice>{message}</Notice>
          {google && (
            <div className="google-entry">
              <Button
                variant="secondary"
                type="button"
                busy={busy}
                onClick={withGoogle}
              >
                <svg
                  width="19"
                  height="19"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path
                    fill="#4285F4"
                    d="M21.6 12.23c0-.71-.06-1.39-.18-2.05H12v3.88h5.38a4.6 4.6 0 0 1-2 3.02v2.51h3.24c1.9-1.75 2.98-4.33 2.98-7.36Z"
                  />
                  <path
                    fill="#34A853"
                    d="M12 22c2.7 0 4.96-.9 6.62-2.41l-3.24-2.51c-.9.6-2.05.97-3.38.97-2.6 0-4.82-1.76-5.61-4.13H3.04v2.59A10 10 0 0 0 12 22Z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M6.39 13.92a6 6 0 0 1 0-3.84V7.49H3.04a10 10 0 0 0 0 9.02l3.35-2.59Z"
                  />
                  <path
                    fill="#EA4335"
                    d="M12 5.95c1.47 0 2.79.5 3.82 1.49l2.87-2.87A9.6 9.6 0 0 0 12 2a10 10 0 0 0-8.96 5.49l3.35 2.59C7.18 7.71 9.4 5.95 12 5.95Z"
                  />
                </svg>
                Continuar con Google
              </Button>
              <p>O utiliza tu correo y contraseña</p>
            </div>
          )}
          <form onSubmit={submit}>
            {mode === "signup" && (
              <label>
                Tu nombre
                <input
                  name="name"
                  required
                  minLength={2}
                  maxLength={100}
                  autoComplete="name"
                  placeholder="¿Cómo te llamas?"
                />
              </label>
            )}
            {mode !== "reset" && (
              <label>
                Correo electrónico
                <input
                  name="email"
                  type="email"
                  required
                  autoComplete="email"
                  placeholder="tu@empresa.com"
                />
              </label>
            )}
            {mode !== "forgot" && (
              <label>
                {mode === "reset" ? "Nueva contraseña" : "Contraseña"}
                <input
                  name="password"
                  type="password"
                  required
                  minLength={mode === "login" ? 1 : 12}
                  maxLength={72}
                  autoComplete={
                    mode === "login" ? "current-password" : "new-password"
                  }
                  placeholder={
                    mode === "login"
                      ? "Tu contraseña"
                      : "Al menos 12 caracteres"
                  }
                />
              </label>
            )}
            <Button busy={busy}>
              {mode === "signup"
                ? "Crear cuenta"
                : mode === "login"
                  ? "Iniciar sesión"
                  : mode === "forgot"
                    ? "Enviar enlace"
                    : "Guardar contraseña"}
              <ArrowRight size={17} />
            </Button>
          </form>
          <div className="auth-links">
            {mode === "login" ? (
              <>
                <a href="/forgot">Olvidé mi contraseña</a>
                <span>
                  ¿Primera vez? <a href={"/signup" + planQuery}>Crea tu cuenta</a>
                </span>
              </>
            ) : (
              <a href={"/login" + planQuery}>Ya tengo cuenta. Iniciar sesión</a>
            )}
          </div>
          <p className="auth-legal">
            Al crear una cuenta aceptas las{" "}
            <a href={`${siteURL}/legal`}>
              condiciones de servicio y privacidad
            </a>
            .
          </p>
        </div>
        <aside className="auth-art">
          <svg viewBox="0 0 500 600" aria-hidden="true">
            <path d="M-40 450H160Q230 450 230 370V240Q230 160 310 160H550" />
            <path d="M-40 200H80Q130 200 130 260V490Q130 550 190 550H550" />
            <circle cx="230" cy="330" r="22" />
            <circle cx="310" cy="160" r="12" />
          </svg>
          <div>
            <span>Una ficha es un comienzo.</span>
            <strong>
              Haz que
              <br />
              llegue más lejos.
            </strong>
          </div>
        </aside>
      </main>
    </div>
  );
}
