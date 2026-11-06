import React, { useState } from "react";

export default function Avatar({ name, src, small = false }) {
  const [failedURL, setFailedURL] = useState(null);
  const showPhoto = src && src !== failedURL;
  return (
    <span className={`avatar${small ? " small" : ""}`} aria-hidden="true">
      {showPhoto ? (
        <img src={src} alt="" referrerPolicy="no-referrer" onError={() => setFailedURL(src)} />
      ) : (
        name?.trim().charAt(0).toLocaleUpperCase() || "?"
      )}
    </span>
  );
}
