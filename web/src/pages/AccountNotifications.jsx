import React from "react";
export default function AccountNotifications() {
  return (
    <section className="docs-section" id="docs-notificaciones">
      <h2>Correos de tu cuenta</h2>
      <p>
        Al crear una cuenta por correo o Google recibes una bienvenida. Iniciar
        sesión o vincular una cuenta existente no vuelve a enviarla. Los
        administradores activos reciben un aviso de los nuevos registros.
      </p>
      <p>
        También te avisamos cuando una suscripción se activa, cambia de plan, se
        renueva, programa o revierte una cancelación, finaliza o tiene un pago
        pendiente. La cancelación programada mantiene el acceso hasta la fecha
        informada; el plan actual siempre se puede consultar en Facturación.
      </p>
      <p>
        Los mensajes salen desde el dominio de Vitruko y no incluyen tu
        contraseña ni API keys. Para recuperar la contraseña recibes un enlace
        de un solo uso que caduca en 30 minutos. Estos correos son
        notificaciones del servicio; no añaden peticiones a tu cuota de API.
      </p>
    </section>
  );
}
