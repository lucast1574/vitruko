import React from "react";
import { gsap } from "gsap";
import "./brand-arrival.css";

const sessionKey = "vitruko-brand-arrival-v1";

export default function BrandArrival() {
  return (
    <div className="brand-arrival" aria-hidden="true">
      <div className="arrival-identity">
        <svg className="arrival-symbol" viewBox="0 0 80 80" fill="none">
          <g transform="translate(8 6)">
            <path
              className="arrival-pin-fill"
              d="M8 3h28l22 22v34H8Z"
            />
            <path
              className="arrival-pin-line"
              d="M8 3h28l22 22v34H8Z"
            />
            <path
              className="arrival-letter"
              d="m19 27 12 20 16-30"
            />
          </g>
        </svg>
        <div className="arrival-name-mask">
          <div className="arrival-name">
            vitruko<span>.</span>
          </div>
        </div>
        <p className="arrival-caption">
          Cada enlace tiene algo que contar.
        </p>
      </div>
    </div>
  );
}

// The overlay is hidden in CSS until animation is available. It never gates
// rendering, data fetching or navigation, and never appears on deep links.
export function createBrandArrival(root) {
  if (location.hash || window.scrollY > 16) return null;
  try {
    if (sessionStorage.getItem(sessionKey)) return null;
    sessionStorage.setItem(sessionKey, "seen");
  } catch {
    /* Storage may be unavailable; the short entrance still works. */
  }
  const overlay = root.querySelector(".brand-arrival");
  const outline = root.querySelector(".arrival-pin-line");
  const letter = root.querySelector(".arrival-letter");
  for (const path of [outline, letter]) {
    const length = path.getTotalLength();
    gsap.set(path, { strokeDasharray: length, strokeDashoffset: length });
  }
  gsap.set(overlay, { display: "grid" });
  return gsap
    .timeline({ defaults: { ease: "power3.out" } })
    .fromTo(
      ".arrival-symbol",
      { y: 16, scale: 0.88 },
      { y: 0, scale: 1, duration: 0.9 },
      0,
    )
    .to(
      outline,
      { strokeDashoffset: 0, duration: 0.8, ease: "power2.inOut" },
      0.05,
    )
    .to(
      letter,
      { strokeDashoffset: 0, duration: 0.55, ease: "power2.inOut" },
      0.3,
    )
    .fromTo(
      ".arrival-name",
      { yPercent: 110 },
      { yPercent: 0, duration: 0.65 },
      0.45,
    )
    .fromTo(
      ".arrival-caption",
      { opacity: 0, y: 8 },
      { opacity: 1, y: 0, duration: 0.45 },
      0.72,
    )
    .to(".arrival-pin-fill", { opacity: 1, duration: 0.3 }, 1.02)
    .to(letter, { stroke: "#202b25", duration: 0.3 }, 1.02)
    .to(
      ".arrival-identity",
      { y: -24, duration: 0.65, ease: "power3.inOut" },
      1.45,
    )
    .to(overlay, { yPercent: -100, duration: 0.65, ease: "power3.inOut" }, 1.45)
    .set(overlay, { display: "none" });
}
