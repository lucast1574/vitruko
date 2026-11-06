import React from "react";
import { Logo } from "../ui";
export default function Legal() {
  return (
    <div className="legal-page">
      <Logo />
      <main>
        <h1>Condiciones y privacidad</h1>
        <p>
          Vitruko es un servicio de API de extracción de productos operado por el titular de
          Nexode. Contacto: <a href="mailto:hola@nexode.app">hola@nexode.app</a>
          . Este aviso se actualizó el 6 de noviembre de 2026.
        </p>
        <h2>Cuenta y uso</h2>
        <p>
          Eres responsable de proteger tu contraseña y tus API keys. No utilices
          el servicio para actividades ilegales, extracción masiva abusiva de
          datos o solicitudes que comprometan su disponibilidad. Podemos
          suspender accesos ante abuso. No envíes datos personales sensibles en
          las consultas.
        </p>
        <h2>Planes y suscripciones</h2>
        <p>
          Free incluye 1.000 peticiones por mes calendario UTC. Launch incluye
          25.000 por USD 9 al mes. Scale cuesta USD 29 al mes y no tiene cuota
          mensual. Sus límites por minuto son 30, 120 y 300, respectivamente.
          Cada petición autenticada a la ruta de extracción admitida por la clave
          consume una unidad, incluso desde caché o si devuelve un error. Los
          rechazos por autenticación, permisos o cuota no cuentan. No cobramos
          excedentes automáticamente.
        </p>
        <p>
          Los planes de pago se renuevan mensualmente mediante Stripe hasta su
          cancelación. Puedes cancelar desde el portal de facturación y
          conservar el acceso hasta el fin del período pagado. Los cambios de
          plan pueden incluir prorrateos calculados por Stripe. Revisa siempre
          el importe y los términos mostrados por Stripe antes de confirmar.
          Contacta al soporte para errores de cobro o solicitudes de devolución;
          se respetan los derechos obligatorios que correspondan.
        </p>
        <h2>Datos y disponibilidad</h2>
        <p>Vitruko consulta fichas públicas de tiendas: Open Graph, JSON-LD, microdatos y fuentes públicas compatibles de Shopify y VTEX. La cobertura puede ser incompleta. No ejecuta JavaScript, no inicia sesión en las tiendas ni supera bloqueos o CAPTCHA. Una respuesta no garantiza inventario, precio ni exactitud comercial. Revisa la información antes de utilizarla.</p>
        <h2>Información que conservamos</h2>
        <p>Guardamos nombre, correo, hash de contraseña, sesiones, hashes de API keys, suscripciones y registros de operación, estado HTTP y tiempo de respuesta. La caché de fichas dura hasta cinco minutos en memoria y no se persiste en el historial personal. El enlace se envía a la tienda que estás consultando; no envíes URLs privadas, secretos ni datos personales sensibles.</p>
        <p>Stripe procesa los pagos sin que Vitruko almacene números de tarjeta. Resend entrega correos de recuperación. Si utilizas Google, Firebase verifica tu identidad y recibimos identificador, nombre, foto y correo verificado para vincular tu cuenta. La sesión propia usa una cookie HttpOnly. No conservamos credenciales de Google en el navegador ni usamos seguimiento publicitario.</p>
        <h2>Contenido y derechos</h2>
        <p>Los textos, imágenes, marcas y otros materiales pertenecen a sus respectivos titulares. Vitruko facilita la lectura de información pública y no concede licencias sobre esos materiales. Eres responsable de respetar las condiciones de las fuentes y contar con los permisos necesarios para su uso o republicación. Las imágenes se devuelven como enlaces externos; mostrarlas puede conectar al visitante con la tienda.</p>
        <h2>Tu cuenta y tus datos</h2>
        <p>Para solicitar acceso, corrección o eliminación de tus datos, escribe desde el correo de tu cuenta a hola@nexode.app. El historial de peticiones se conserva hasta 180 días para permitir su consulta y atender incidencias. Podemos conservar los datos que exijan las obligaciones legales y de facturación aplicables.</p>
        <a href="/">Volver a Vitruko</a>
      </main>
    </div>
  );
}
