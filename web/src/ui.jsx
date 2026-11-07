import React from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { ChevronDown, Check, X, ArrowUpRight } from "lucide-react";
export function Logo({ large = false }) {
  return (
    <a
      className={`brand ${large ? "brand-large" : ""}`}
      href="/"
      aria-label="Vitruko, inicio"
    >
      <img src="/logo.svg?v=tag-2" alt="" width="36" height="36" />
      <span>
        vitruko<span className="brand-dot">.</span>
      </span>
    </a>
  );
}
export function Button({ children, variant = "", busy = false, ...props }) {
  return (
    <button
      className={`button ${variant}`}
      disabled={busy || props.disabled}
      {...props}
    >
      {busy ? <span className="loading-dot" /> : null}
      {children}
    </button>
  );
}
export function Notice({ children, error = false }) {
  return children ? (
    <div
      role={error ? "alert" : "status"}
      className={`notice ${error ? "error" : ""}`}
    >
      {children}
    </div>
  ) : null;
}
export function Empty({ title, children, action }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      <p>{children}</p>
      {action}
    </div>
  );
}
export function Select({ value, onChange, options, label }) {
  const [trigger, setTrigger] = React.useState(null);
  // Native modal dialogs make body-level portals inert and paint above them.
  // Keep the options in the same dialog as their trigger.
  const portalContainer = trigger?.closest("dialog") || undefined;
  return (
    <SelectPrimitive.Root value={value} onValueChange={onChange}>
      <SelectPrimitive.Trigger
        ref={setTrigger}
        className="select-trigger"
        aria-label={label}
      >
        <SelectPrimitive.Value />
        <SelectPrimitive.Icon>
          <ChevronDown size={16} />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectPrimitive.Portal container={portalContainer}>
        <SelectPrimitive.Content
          position="popper"
          sideOffset={5}
          className="select-content"
        >
          <SelectPrimitive.Viewport>
            {options.map((o) => (
              <SelectPrimitive.Item
                className="select-item"
                value={o.value}
                key={o.value}
              >
                <SelectPrimitive.ItemText>{o.label}</SelectPrimitive.ItemText>
                <SelectPrimitive.ItemIndicator>
                  <Check size={16} />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
export function Modal({ title, children, onClose }) {
  const ref = React.useRef(null);
  React.useEffect(() => {
    ref.current.showModal();
    const el = ref.current;
    const onClick = (e) => {
      if (e.target === el) onClose();
    };
    el.addEventListener("click", onClick);
    return () => el.removeEventListener("click", onClick);
  }, []);
  return (
    <dialog
      className="modal"
      ref={ref}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      <div className="modal-heading">
        <h2>{title}</h2>
        <button className="icon-button" onClick={onClose} aria-label="Cerrar">
          <X size={20} />
        </button>
      </div>
      {children}
    </dialog>
  );
}
export function External({ children, ...props }) {
  return (
    <a {...props} target="_blank" rel="noreferrer">
      {children}
      <ArrowUpRight size={15} />
    </a>
  );
}
