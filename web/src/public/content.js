export const siteOrigin = "https://vitruko.nexode.app";
export const pages = {
  "/": {
    title: "Vitruko | API de extracción de productos y enlaces",
    description:
      "Convierte enlaces de tiendas en fichas de producto JSON. Marca, galería, precio, disponibilidad y calidad de datos con API keys, cuotas y panel. Empieza con 1.000 peticiones gratis.",
  },
  "/documentacion": {
    title: "Documentación de la API de productos | Vitruko",
    description:
      "Integra POST /v1/extract: autenticación, respuesta JSON, caché, cuotas y errores. Prueba un enlace de producto con tu propia API key.",
    heading: "Documentación de Vitruko",
  },
  "/legal": {
    title: "Condiciones y privacidad | Vitruko",
    description:
      "Condiciones de uso, planes, suscripciones y tratamiento de datos de Vitruko.",
  },
  "/api-extraer-productos": {
    title: "API para extraer información de productos | Vitruko",
    description:
      "Extrae nombre, marca, galería, precio y disponibilidad publicados por tiendas en una ficha JSON con calidad y procedencia.",
    heading: "Extrae la ficha. Construye lo demás.",
    navTitle: "Extracción de productos",
    summary:
      "Extrae nombre, marca, galería, precio y disponibilidad publicados por tiendas en una ficha JSON con calidad y procedencia.",
    scope: "extract",
    operation: "extract",
    code: 'curl "https://api-vitruko.nexode.app/v1/extract" \\\n  -H "X-API-Key: TU_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d \'{"url":"https://tienda.com/products/articulo"}\'',
    sections: [
      {
        title: "Un contrato para tu interfaz",
        text: "Una lista de regalos o una colección de productos empieza con un enlace. Copiar a mano nombre, descripción e imagen añade trabajo y errores. Vitruko recibe la URL y devuelve una ficha JSON. Puedes asignarlos a tus componentes, revisarlos y guardarlos en tu propia aplicación. La API también devuelve marca, galería, precio con moneda y disponibilidad cuando la fuente los publica, junto con datos de calidad. No estima los valores ausentes.",
      },
      {
        title: "Cómo se elige la información",
        text: "El extractor lee el HTML público de la página. Considera Open Graph, metadatos de Twitter, el título del documento, JSON-LD de tipo Product o ProductGroup y microdatos de producto. Cuando encuentra un producto estructurado, su nombre tiene prioridad sobre el título genérico de la tienda y su imagen sobre un logotipo. JSON-LD también puede estar dentro del cuerpo del documento; la lectura está acotada para no procesar respuestas arbitrariamente grandes.",
      },
      {
        title: "El enlace identifica el artículo",
        text: "La normalización elimina parámetros de seguimiento conocidos, como utm_source o gclid. Conserva parámetros de variante y vendedor. En enlaces compartidos de Mercado Libre, mantiene pdp_filters e intenta preservar el artículo que se encuentre en el fragmento. No sustituye la ficha por un resultado de búsqueda ni por otro producto que tenga una imagen más fácil de obtener.",
      },
      {
        title: "Integra la respuesta con revisión",
        text: "Una respuesta HTTP 200 incluye title, description, imageUrl y url. Los campos de descripción e imagen pueden estar vacíos. No presentes esos vacíos como un fallo total ni inventes un valor; ofrece una edición manual. Si la página no publica una ficha útil o bloquea la consulta, HTTP 422 permite conservar la URL y pedir los datos necesarios. La disponibilidad de las fuentes externas puede cambiar sin aviso.",
      },
    ],
  },
  "/vista-previa-enlaces": {
    title: "API de vistas previas de enlaces de productos | Vitruko",
    description:
      "Muestra una vista previa del producto con la información pública de la tienda y conserva el enlace original normalizado.",
    heading: "Un enlace que ya dice algo.",
    navTitle: "Vistas previas de enlaces",
    summary:
      "Muestra una vista previa del producto con la información pública de la tienda y conserva el enlace original normalizado.",
    scope: "extract",
    operation: "extract",
    code: 'curl "https://api-vitruko.nexode.app/v1/extract" \\\n  -H "X-API-Key: TU_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d \'{"url":"https://tienda.com/products/articulo"}\'',
    sections: [
      {
        title: "De la URL a una vista legible",
        text: "Una URL por sí sola no muestra qué eligió una persona. La respuesta de Vitruko permite construir una vista previa con título, descripción e imagen. Úsala en listas de deseos, regalos, comparadores que gestionen sus propios datos o colecciones personales. Vitruko entrega información; el diseño y la decisión de guardar el producto pertenecen a tu aplicación.",
      },
      {
        title: "Imágenes sin copiar credenciales",
        text: "imageUrl es una URL pública de imagen obtenida de los metadatos. Los enlaces relativos se resuelven contra la página consultada y se descartan esquemas o destinos locales evidentes. No se transfieren cookies del comprador ni credenciales de Vitruko a la tienda. Si el navegador carga esa imagen, contacta directamente con el servidor de origen; utiliza referrerpolicy no-referrer cuando corresponda y una política de imágenes adecuada.",
      },
      {
        title: "Diseña para respuestas incompletas",
        text: "No todas las tiendas publican Open Graph o JSON-LD completo. Muestra un bloque neutral cuando imageUrl esté vacío, limita el texto de descripción y deja que la persona confirme el título antes de guardar. Trata la salida como datos, nunca como HTML ejecutable: renderiza el texto con las protecciones normales de tu framework. El endpoint devuelve el precio y la disponibilidad publicados cuando puede identificarlos, sin garantía de inventario actual. price es null y availability es unknown si no se pudieron confirmar.",
      },
      {
        title: "Repeticiones y cambios en la tienda",
        text: "Las respuestas correctas pueden permanecer cinco minutos en caché en memoria. X-Cache indica HIT o MISS. La clave de caché conserva el identificador de variante para no mezclar enlaces de productos diferentes. Cada llamada autenticada admitida consume una petición, también cuando usa caché. Una modificación en la tienda puede tardar en reflejarse; si necesitas datos contractuales de compra, compruébalos en la fuente.",
      },
    ],
  },
  "/shopify-vtex": {
    title: "Extraer productos de Shopify, VTEX y Mercado Libre | Vitruko",
    description:
      "Conecta fichas públicas de distintas plataformas sin mantener un contrato diferente por tienda.",
    heading: "Tiendas distintas. La misma respuesta.",
    navTitle: "Shopify, VTEX y Mercado Libre",
    summary:
      "Conecta fichas públicas de distintas plataformas sin mantener un contrato diferente por tienda.",
    scope: "extract",
    operation: "extract",
    code: 'curl "https://api-vitruko.nexode.app/v1/extract" \\\n  -H "X-API-Key: TU_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d \'{"url":"https://tienda.com/products/articulo"}\'',
    sections: [
      {
        title: "Primero, la información de la página",
        text: "El primer paso es común: consultar la URL HTTPS pública y leer sus metadatos. Vitruko no necesita que la persona identifique la plataforma antes de enviar el enlace. Cuando existe una ficha útil en el HTML, puede devolverla directamente. No hay integración con cuentas privadas de comerciantes ni lectura de pedidos o datos de compradores.",
      },
      {
        title: "Complementos públicos de Shopify y VTEX",
        text: "Para rutas products compatibles, el extractor consulta primero la API pública de producto de Shopify, conservando prefijos de idioma. Para VTEX puede complementar el HTML con su fuente pública de catálogo cuando identifica una página compatible terminada en p. Estos complementos usan el mismo cliente que verifica destinos públicos. En VTEX se comprueba la coincidencia del identificador de la ficha para evitar devolver otro artículo del catálogo.",
      },
      {
        title: "Enlaces compartidos de Mercado Libre",
        text: "Los enlaces compartidos pueden contener un identificador de artículo además del producto agrupado. Vitruko elimina seguimiento conocido y mantiene el identificador seleccionado. El perfil de vista previa se limita a dominios reales de Mercado Libre; no se aplica a hosts que solo incluyen ese nombre como parte de otro dominio. El resultado depende de la información pública disponible en el momento de consultar.",
      },
      {
        title: "Lo que debes prever en producción",
        text: "Una tienda puede restringir solicitudes automatizadas, mover sus rutas o exigir JavaScript. Vitruko no ejecuta un navegador ni resuelve CAPTCHA; esos casos pueden devolver HTTP 422. Maneja el error con edición manual y conserva el enlace. Si la tienda impone una espera, HTTP 429 con code source_rate_limited y Retry-After indica cuándo reintentar. Las consultas al mismo dominio se espacian y durante esa espera no se vuelve a consultar la tienda. Respeta los límites de tu plan y las condiciones de cada fuente. Los nombres de plataformas describen compatibilidad técnica y no una asociación comercial.",
      },
    ],
  },
};
export const guides = Object.entries(pages).filter(
  ([, page]) => page.operation,
);
export const notFound = {
  title: "Página no encontrada | Vitruko",
  description: "La página que buscas no está disponible.",
};
