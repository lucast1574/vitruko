import { initializeApp, getApps } from "@firebase/app";
import {
  initializeAuth,
  browserPopupRedirectResolver,
  GoogleAuthProvider,
  inMemoryPersistence,
  signInWithPopup,
  signOut,
} from "@firebase/auth";

export async function googleIdentity(config) {
  const app =
    getApps().find((a) => a.name === "vitruko-google") ||
    initializeApp(config, "vitruko-google");
  const auth = initializeAuth(app, {
    persistence: inMemoryPersistence,
    popupRedirectResolver: browserPopupRedirectResolver,
  });
  auth.languageCode = "es";
  const provider = new GoogleAuthProvider();
  provider.setCustomParameters({ prompt: "select_account" });
  let timeout;
  try {
    const result = await Promise.race([
      signInWithPopup(auth, provider),
      new Promise((_, reject) => {
        timeout = setTimeout(
          () =>
            reject(
              new Error(
                "El acceso con Google tardó demasiado. Cierra su ventana e intenta de nuevo.",
              ),
            ),
          120000,
        );
      }),
    ]);
    return await result.user.getIdToken();
  } finally {
    // Vitruko keeps its own HttpOnly session; do not persist Google credentials.
    clearTimeout(timeout);
    void signOut(auth).catch(() => {});
  }
}
