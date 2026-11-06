import React from "react";
import Inspector from "./Inspector";
export default function Docs({
  heading = "Documentación de la API de Vitruko",
}) {
  return (
    <div className="docs-page">
      <div className="page-title">
        <div>
          <h1>{heading}</h1>
          <p>Un enlace público. Una respuesta JSON lista para integrar.</p>
        </div>
        <a
          className="text-link"
          href="https://api-vitruko.nexode.app/openapi.json"
          target="_blank"
          rel="noreferrer"
        >
          OpenAPI
        </a>
      </div>
      <section className="docs-auth">
        <h2>Extraer un producto</h2>
        <p>
          Base URL: <code>https://api-vitruko.nexode.app</code>. Envía{" "}
          <code>POST /v1/extract</code>, un cuerpo JSON con <code>url</code> y
          tu clave en <code>X-API-Key</code>. También se admite{" "}
          <code>Authorization: Bearer TU_API_KEY</code>.
        </p>
        <pre>{`curl "https://api-vitruko.nexode.app/v1/extract" \\
  -H "X-API-Key: TU_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"url":"https://tienda.com/products/articulo"}'`}</pre>
        <p>
          La URL debe ser HTTPS pública, sin credenciales ni puertos
          personalizados, de hasta 2.048 caracteres. Tu clave necesita el
          permiso <code>extract</code>. Nunca envíes la clave en la URL ni la
          publiques en un repositorio.
        </p>
      </section>
      <section>
        <h2>Contrato de respuesta</h2>
        <pre>{`{
  "title": "Nombre del producto",
  "imageUrl": "https://tienda.com/imagen.jpg",
  "description": "Descripción publicada por la tienda",
  "url": "https://tienda.com/products/articulo"
}`}</pre>
        <p>
          <code>title</code> identifica la ficha. <code>imageUrl</code> y{" "}
          <code>description</code> pueden estar vacíos. <code>url</code>{" "}
          conserva el enlace normalizado: elimina seguimiento conocido y
          mantiene identificadores de producto y variante. No incluye precio,
          inventario ni datos privados.
        </p>
      </section>
      <section>
        <h2>Prueba una petición real</h2>
        <p>
          Utiliza tu propia clave. Solo se consulta la tienda cuando pulsas
          «Extraer producto». Esta operación consume una petición de tu cuenta.
        </p>
        <Inspector embedded />
      </section>
      <section>
        <h2>Cuotas y caché</h2>
        <p>
          Las cabeceras <code>X-Quota-Limit</code>,{" "}
          <code>X-Quota-Remaining</code> y <code>X-Quota-Reset</code> indican tu
          cuota. Para Scale, las primeras dos indican <code>unlimited</code>. El
          período Free es el mes UTC; los planes pagados usan su período de
          facturación.
        </p>
        <p>
          <code>X-Cache: HIT</code> indica una respuesta en memoria de hasta
          cinco minutos; <code>MISS</code>, una consulta nueva. Ambas cuentan
          una petición. Los errores posteriores a admitir la clave también
          cuentan. Los rechazos de clave, permisos o cuota no se cobran. Respeta{" "}
          <code>Retry-After</code> cuando la API lo indique.
        </p>
      </section>
      <section>
        <h2>Errores que debes manejar</h2>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>HTTP</th>
                <th>Qué significa</th>
                <th>Qué hacer</th>
              </tr>
            </thead>
            <tbody>
              {[
                [
                  400,
                  "URL o JSON inválido",
                  "Revisa url y envía un solo objeto JSON.",
                ],
                [401, "Clave ausente o revocada", "Comprueba X-API-Key."],
                [
                  403,
                  "Permiso faltante o cuenta suspendida",
                  "Usa una clave con extract y una cuenta activa.",
                ],
                [405, "Método incorrecto", "Usa POST."],
                [
                  415,
                  "Formato incorrecto",
                  "Envía Content-Type: application/json.",
                ],
                [
                  422,
                  "Ficha no accesible",
                  "Conserva el enlace y ofrece edición manual.",
                ],
                [
                  429,
                  "Cuota o velocidad excedida",
                  "Respeta Retry-After o cambia de plan.",
                ],
                [
                  503,
                  "Servicio ocupado o renovación pendiente",
                  "Reintenta con espera; revisa tu facturación.",
                ],
              ].map(([code, meaning, action]) => (
                <tr key={code}>
                  <td>{code}</td>
                  <td>{meaning}</td>
                  <td>{action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section>
        <h2>Alcance y uso de los datos</h2>
        <p>
          Vitruko lee Open Graph, JSON-LD y microdatos; complementa fichas
          compatibles con fuentes públicas de Shopify y VTEX. La consulta tiene
          un límite de tiempo de doce segundos. No ejecuta JavaScript, no usa
          sesiones de comprador ni supera CAPTCHA o bloqueos de tiendas. La
          disponibilidad puede variar.
        </p>
        <p>
          Los nombres, imágenes y descripciones pertenecen a sus titulares.
          Comprueba las condiciones de la fuente y tus permisos antes de
          republicar información. Las imágenes se devuelven como URL; mostrarlas
          puede conectar el navegador del visitante con la tienda. Revisa los
          datos antes de guardarlos como definitivos.
        </p>
      </section>
    </div>
  );
}
