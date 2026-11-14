import React, { useEffect, useRef, useState } from "react";
import { ArrowRight, ArrowUpRight, Check } from "lucide-react";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { Logo } from "./ui";
import { appURL } from "./client";
import { useData } from "./useData";
import BrandArrival, { createBrandArrival } from "./BrandArrival";
import { ResourceLinks } from "./public/PublicPage";

const examples = [
  {
    title: "Taza de cerámica",
    description: "Una pausa, bien servida. Cerámica esmaltada de 300 ml.",
    imageUrl: "/objects/cup.svg",
    url: "https://tienda.example/products/taza?variant=natural",
  },
  {
    title: "Lámpara de mesa",
    description: "Luz para tus ideas. Pantalla de metal y base de madera.",
    imageUrl: "/objects/lamp.svg",
    url: "https://tienda.example/products/lampara",
  },
  {
    title: "Bolso de lona",
    description: "Lo esencial, contigo. Algodón resistente con asas largas.",
    imageUrl: "/objects/bag.svg",
    url: "https://tienda.example/products/bolso",
  },
];
const faqs = [
  [
    "¿Funciona con cualquier tienda?",
    "Puedes enviar una URL HTTPS pública. La extracción depende de los datos que publique la tienda: Open Graph, JSON-LD, microdatos y las fuentes públicas compatibles de Shopify y VTEX. Si exige iniciar sesión, bloquea la consulta o solo genera la ficha con JavaScript, puede devolver HTTP 422. No prometemos cobertura universal.",
  ],
  [
    "¿Qué información devuelve?",
    "Nombre, descripción, marca, galería, enlace normalizado, precio con moneda y disponibilidad cuando la fuente los publica. quality indica los datos faltantes y el origen de cada campo. Sin precio y moneda identificables, price es null; nunca inventamos el dato.",
  ],
  [
    "¿Qué cuenta como una petición?",
    "Cada llamada admitida por una API key consume una petición, también desde caché o si la tienda devuelve un error. Las claves inválidas, los permisos rechazados y las llamadas bloqueadas por cuota no consumen unidades. El panel y sus visitas no cuentan.",
  ],
  [
    "¿Qué significa ilimitado?",
    "Scale no tiene cuota mensual. Mantiene 300 peticiones por minuto y la capacidad concurrente del servicio. No hay cargos automáticos por excedentes. Las tiendas conservan sus propias condiciones y límites.",
  ],
  [
    "¿Se conservan mis enlaces?",
    "La caché de respuestas dura hasta cinco minutos en memoria. Tu historial muestra operación, estado y tiempo de respuesta; no guarda las URLs consultadas. No enviamos cookies de usuario ni claves a las tiendas.",
  ],
];
export default function Landing() {
  const root = useRef(null),
    [selected, setSelected] = useState(0);
  const { data } = useData("/console/plans");
  const item = examples[selected];
  useEffect(() => {
    gsap.registerPlugin(ScrollTrigger);
    const media = gsap.matchMedia();
    media.add("(prefers-reduced-motion: no-preference)", () => {
      const arrival = createBrandArrival(root.current);
      let entrance;
      const context = gsap.context(() => {
        entrance = gsap
          .timeline({ delay: arrival ? 1.7 : 0.1 })
          .from(
            ".v-hero h1, .v-hero .hero-lead, .v-hero .hero-cta",
            { y: 36, opacity: 0, stagger: 0.12, duration: 0.9 },
            0,
          )
          .from(
            ".v-showcase",
            {
              clipPath: "inset(0 100% 0 0)",
              duration: 1.1,
              ease: "power3.inOut",
            },
            0.05,
          );
        gsap.utils
          .toArray(".v-reveal")
          .forEach((el) =>
            gsap.from(el, {
              y: 34,
              opacity: 0,
              duration: 0.8,
              scrollTrigger: { trigger: el, start: "top 92%", once: true },
            }),
          );
        gsap.to(".v-object img", {
          y: -15,
          rotation: 3,
          duration: 3,
          yoyo: true,
          repeat: -1,
          ease: "sine.inOut",
        });
        gsap.to(".data-sheet", {
          y: -32,
          ease: "none",
          scrollTrigger: {
            trigger: ".v-showcase",
            start: "top top",
            end: "bottom top",
            scrub: 1,
          },
        });
      }, root);
      const dismiss = () => {
        if (arrival && arrival.progress() < 1) {
          arrival.progress(1);
          entrance?.progress(1);
          gsap.set(".v-hero h1, .hero-lead, .hero-cta, .v-showcase", {
            clearProps: "all",
          });
        }
      };
      window.addEventListener("keydown", dismiss);
      window.addEventListener("pointerdown", dismiss);
      window.addEventListener("wheel", dismiss, { passive: true });
      return () => {
        window.removeEventListener("keydown", dismiss);
        window.removeEventListener("pointerdown", dismiss);
        window.removeEventListener("wheel", dismiss);
        arrival?.kill();
        context.revert();
      };
    });
    return () => media.revert();
  }, []);
  return (
    <div className="v-landing" ref={root}>
      <BrandArrival />
      <header className="v-nav">
        <Logo />
        <nav aria-label="Principal">
          <a href="#producto">Producto</a>
          <a href="#planes">Planes</a>
          <a href="/documentacion">Documentación</a>
        </nav>
        <a href={`${appURL}/login`}>
          Mi cuenta <ArrowUpRight size={17} />
        </a>
      </header>
      <main>
        <section className="v-hero">
          <div className="hero-title">
            <h1>
              De enlace
              <br />a producto.
            </h1>
            <p className="hero-lead">
              La API que convierte una página de venta en datos que tu
              aplicación entiende. Nombre, galería, marca, precio y disponibilidad
              publicados por la tienda, en una petición.
            </p>
          </div>
          <div className="hero-cta">
            <a className="button" href={`${appURL}/signup`}>
              Crea tu API key <ArrowRight size={18} />
            </a>
            <span>
              1.000 peticiones al mes, gratis.
              <br />
              Sin tarjeta para empezar.
            </span>
          </div>
          <div className="v-showcase" data-ui>
            <div className="shop-window">
              <div className="window-top">
                <span>Una ficha de ejemplo</span>
                <span>Vitruko studio</span>
              </div>
              <div className="v-object">
                <img
                  src={item.imageUrl}
                  width="520"
                  height="420"
                  alt={item.title}
                />
              </div>
              <div className="shop-caption">
                <strong>{item.title}</strong>
                <div
                  className="object-tabs"
                  role="tablist"
                  aria-label="Elegir producto de ejemplo"
                  onKeyDown={(e) => {
                    const direction =
                      e.key === "ArrowRight"
                        ? 1
                        : e.key === "ArrowLeft"
                          ? -1
                          : 0;
                    if (direction) {
                      e.preventDefault();
                      const next =
                        (selected + direction + examples.length) %
                        examples.length;
                      setSelected(next);
                      e.currentTarget.querySelectorAll("button")[next].focus();
                    }
                  }}
                >
                  {examples.map((e, i) => (
                    <button
                      key={e.title}
                      role="tab"
                      tabIndex={selected === i ? 0 : -1}
                      aria-selected={selected === i}
                      onClick={() => {
                        setSelected(i);
                        }}
                    >
                      {["Taza", "Lámpara", "Bolso"][i]}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="data-sheet" aria-live="polite">
              <div className="sheet-top">
                <span>De enlace a ficha</span>
                <Check size={18} aria-hidden="true" />
              </div>
              <div className="sheet-product">
                <span className="sheet-brand">Vitruko studio</span>
                <h2>{item.title}</h2>
                <p>{item.description}</p>
              </div>
              <dl className="sheet-fields">
                <div><dt>Imagen del producto</dt><dd>Lista para mostrar</dd></div>
                <div><dt>Enlace de origen</dt><dd>tienda.example</dd></div>
              </dl>
              <p className="sheet-context">También puedes obtener galería, precio y disponibilidad cuando la tienda los publica.</p>
              <a className="sheet-docs" href="/documentacion">Ver la respuesta de la API <ArrowUpRight size={17} /></a>
              <div className="sheet-bottom"><span>Ficha ilustrativa</span><span>Tu interfaz, tus reglas.</span></div>
            </div>
          </div>
        </section>
        <section className="v-statement v-reveal" id="producto">
          <h2>
            La ficha.
            <br />
            Sin copiar la ficha.
          </h2>
          <div>
            <p>
              Tu usuario pega un enlace. Tu producto recibe los datos. Así de
              directo para listas de regalos, catálogos, colecciones y vistas
              previas.
            </p>
            <a href="/documentacion" className="text-link">
              Explora la integración <ArrowRight size={19} />
            </a>
          </div>
        </section>
        <section className="v-process v-reveal" id="integracion">
          <div className="process-heading">
            <h2>
              Una URL entra.
              <br />
              Tu interfaz sale.
            </h2>
            <p>
              El mismo contrato para diferentes tiendas. Conserva los
              identificadores de variante y lee los datos que la página
              comparte.
            </p>
          </div>
          <div className="process-content" data-ui>
            <div className="request-slip">
              <h3>Tu petición</h3>
              <pre>{`fetch("https://api.vitruko.lat/v1/extract", {
  method: "POST",
  headers: {
    "X-API-Key": "TU_API_KEY",
    "Content-Type": "application/json"
  },
  body: JSON.stringify({ url: enlace })
})`}</pre>
              <a href="/documentacion">
                Ver parámetros y errores <ArrowUpRight size={16} />
              </a>
            </div>
            <div className="result-slip">
              <h3>Tu siguiente pantalla</h3>
              <div className="mini-product">
                <img
                  src="/objects/cup.svg"
                  alt="Taza de ejemplo"
                  width="150"
                  height="140"
                />
                <div>
                  <strong>Taza de cerámica</strong>
                  <p>Una pausa, bien servida.</p>
                  <span>Vista previa de ejemplo</span>
                </div>
              </div>
              <p>
                Asigna <code>title</code>, <code>imageUrl</code>,{" "}
                <code>description</code> y <code>url</code> a tus componentes. Añade galería, marca, precio y disponibilidad cuando estén publicados; quality indica qué falta y el origen de cada campo.
                Tú decides qué construir.
              </p>
            </div>
          </div>
        </section>
        <section className="v-support v-reveal">
          <h2>
            Distintas vitrinas.
            <br />
            Un mismo idioma.
          </h2>
          <div className="source-list">
            <p>
              <strong>Datos de la página</strong>
              <span>Open Graph, JSON-LD de producto y microdatos.</span>
            </p>
            <p>
              <strong>Shopify y VTEX</strong>
              <span>Fuentes públicas de la ficha cuando hacen falta.</span>
            </p>
            <p>
              <strong>Mercado Libre</strong>
              <span>Enlaces compartidos con el artículo seleccionado.</span>
            </p>
          </div>
          <p className="support-note">
            La disponibilidad depende de cada tienda. No ejecutamos JavaScript
            ni accedemos a contenido privado.
          </p>
        </section>
        <section className="v-account v-reveal">
          <div>
            <h2>
              Las claves contigo.
              <br />
              El consumo, claro.
            </h2>
            <p>
              Crea claves por proyecto, revócalas cuando quieras y consulta las
              peticiones de tu cuenta. Tu facturación y tu acceso de Google
              viven en el mismo panel.
            </p>
            <a className="text-link" href={`${appURL}/signup`}>
              Abre tu cuenta <ArrowRight size={18} />
            </a>
          </div>
          <div className="account-sample" data-ui>
            <div className="sample-key">
              <span>Mi catálogo · API key de ejemplo</span>
              <strong>vt_live_••••••••••••</strong>
              <span>Permiso: extraer productos</span>
            </div>
            <div className="sample-usage">
              <span>Plan Free · Incluido cada mes</span>
              <strong>
                1.000 <small>peticiones</small>
              </strong>
              <p>La cuota se comparte entre las claves de tu cuenta.</p>
            </div>
          </div>
        </section>
        <section className="v-plans v-reveal" id="planes">
          <h2>Crece por peticiones.</h2>
          <p>Tres planes. Sin créditos que descifrar.</p>
          <div className="v-plan-list">
            {(
              data?.plans || [
                {
                  id: "free",
                  name: "Free",
                  monthlyLimit: 1000,
                  rpm: 30,
                  priceCents: 0,
                },
                {
                  id: "launch",
                  name: "Launch",
                  monthlyLimit: 25000,
                  rpm: 120,
                  priceCents: 900,
                },
                {
                  id: "scale",
                  name: "Scale",
                  monthlyLimit: 0,
                  rpm: 300,
                  priceCents: 2900,
                },
              ]
            ).map((p) => (
              <article key={p.id}>
                <h3>{p.name}</h3>
                <div className="plan-price">
                  {p.priceCents ? `$${p.priceCents / 100}` : "$0"}
                  <span>USD / mes</span>
                </div>
                <strong>
                  {p.monthlyLimit
                    ? new Intl.NumberFormat("es-PE").format(p.monthlyLimit) +
                      " peticiones"
                    : "Peticiones ilimitadas"}
                </strong>
                <p>{p.rpm} peticiones por minuto</p>
                <p>Extracción, API keys e historial</p>
                <a className="button" href={`${appURL}/signup?plan=${p.id}`}>
                  {p.id === "free" ? "Empieza gratis" : "Elige " + p.name}
                  <ArrowUpRight size={18} />
                </a>
              </article>
            ))}
          </div>
          <p className="plan-footnote">
            Cada petición admitida cuenta una unidad, también desde caché o con
            error. Scale elimina la cuota mensual; conserva el límite por
            minuto. Sin cargos automáticos por excedentes. Pagos mensuales
            mediante Stripe.
          </p>
        </section>
        <section className="v-guides v-reveal">
          <h2>Construye con contexto.</h2>
          <ResourceLinks />
        </section>
        <section className="v-faq v-reveal">
          <h2>Antes de integrar.</h2>
          <div>
            {faqs.map(([q, a]) => (
              <details key={q}>
                <summary>
                  {q}
                  <span aria-hidden="true">+</span>
                </summary>
                <p>{a}</p>
              </details>
            ))}
          </div>
        </section>
        <section className="v-close">
          <h2>
            Tu próximo producto
            <br />
            empieza con un enlace.
          </h2>
          <a className="button" href={`${appURL}/signup`}>
            Dale forma <ArrowRight size={19} />
          </a>
        </section>
      </main>
      <footer className="v-footer">
        <div>
          <Logo />
          <nav>
            <a href="/documentacion">Documentación</a>
            <a href={`${appURL}/login`}>Mi cuenta</a>
            <a href="/legal">Condiciones y privacidad</a>
            <a href="mailto:hola@nexode.app">Contacto</a>
          </nav>
          <span>Vitruko © {new Date().getFullYear()}</span>
        </div>
        <div className="footer-wordmark" aria-hidden="true">
          vitruko.
        </div>
      </footer>
    </div>
  );
}
