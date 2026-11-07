import React, { useState } from "react";
import { LogOut, Save } from "lucide-react";
import { api } from "../client";
import { Button, Notice } from "../ui";
export default function Profile({ user, onUpdate }) {
  const [message, setMessage] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState("");
  async function save(e) {
    e.preventDefault();
    setBusy("profile");
    setError("");
    setMessage("");
    try {
      const updated = await api("/console/me", "PATCH", {
        name: new FormData(e.currentTarget).get("name"),
      });
      onUpdate(updated);
      setMessage("Perfil actualizado.");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy("");
    }
  }
  async function password(e) {
    e.preventDefault();
    const form = e.currentTarget;
    setBusy("password");
    setError("");
    setMessage("");
    const f = new FormData(form);
    try {
      await api("/console/password", "POST", {
        currentPassword: f.get("currentPassword"),
        newPassword: f.get("newPassword"),
      });
      form.reset();
      setMessage("Contraseña actualizada. Las otras sesiones se cerraron.");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Mi perfil</h1>
          <p>Tu identidad y la seguridad de tu cuenta.</p>
        </div>
      </div>
      <Notice error>{error}</Notice>
      <Notice>{message}</Notice>
      <div className="settings-row">
        <div>
          <h2>Información de la cuenta</h2>
          <p>Tu correo identifica la cuenta y las notificaciones de acceso.</p>
        </div>
        <form onSubmit={save}>
          <label>
            Nombre
            <input
              name="name"
              defaultValue={user.name}
              required
              minLength={2}
              maxLength={100}
            />
          </label>
          <label>
            Correo
            <input value={user.email} readOnly type="email" />
          </label>
          <Button busy={busy === "profile"} disabled={!!busy}>
            <Save size={16} />
            Guardar perfil
          </Button>
        </form>
      </div>
      <div className="settings-row">
        <div>
          <h2>Cambiar contraseña</h2>
          <p>
            Usa una contraseña única, de 12 a 72 caracteres. Si entras con
            Google y aún no tienes una contraseña, puedes{" "}
            <a href="/forgot">definirla por correo</a>.
          </p>
        </div>
        <form onSubmit={password}>
          <label>
            Contraseña actual
            <input
              name="currentPassword"
              type="password"
              autoComplete="current-password"
              required
            />
          </label>
          <label>
            Nueva contraseña
            <input
              name="newPassword"
              type="password"
              autoComplete="new-password"
              required
              minLength={12}
              maxLength={72}
            />
          </label>
          <Button busy={busy === "password"} disabled={!!busy}>Actualizar contraseña</Button>
        </form>
      </div>
      <div className="settings-row">
        <div>
          <h2>Esta sesión</h2>
          <p>Cierra el acceso desde este navegador cuando termines.</p>
        </div>
        <Button
          variant="secondary"
          busy={busy === "logout"}
          disabled={!!busy}
          onClick={async () => {
            setBusy("logout");
            setError("");
            try {
              await api("/console/logout", "POST", {});
              location.assign("/login");
            } catch (e) {
              setError(e.message);
              setBusy("");
            }
          }}
        >
          <LogOut size={16} />
          Cerrar sesión
        </Button>
      </div>
    </>
  );
}
