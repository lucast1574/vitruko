# Validación de flujos

## Checks obligatorios antes de publicar

- `npm --prefix web ci`
- `npm --prefix web test`: cliente HTTP, errores de recarga, sesión caducada, selección del plan y búsquedas fuera de orden.
- `npm --prefix web run build`
- `npm --prefix web audit --omit=dev`
- `TEST_DATABASE_URL=... go test -race -count=1 ./...` contra PostgreSQL 18 aislado con el nombre de base exigido por los tests. Sin esa variable las pruebas de integración se omiten: no confundirlo con validar los flujos de base de datos.
- `go vet ./...` y `go build ./cmd/api`.

La CI ejecuta los checks de frontend y backend con PostgreSQL; deben pasar en el commit publicado.

## Recorrido de interfaz en un navegador real

Las pruebas DOM no simulan la capa superior de un dialog nativo. Comprobar los siguientes pasos en un navegador real, también a 390 px de ancho:

1. Login válido e inválido; error legible y recuperación de los botones. Acceso con Google: cancelación y entrada. Elección de Launch/Scale conservada al alternar registro/login y al entrar a facturación.
2. Resumen y consumo: datos de la cuenta, períodos, peticiones admitidas y errores; estados de carga y fallo.
3. Perfil: editar nombre, guardar y comprobar inmediatamente cabecera/sidebar y persistencia al recargar. Salir solo desde perfil; comprobar que una sesión caducada vuelve al login.
4. Claves con cuenta de prueba: crear con permisos, copiar, consultar listado sin revelar el secreto, límite de diez, revocar y comprobar rechazo posterior. No revocar claves de clientes para probar.
5. Administración: filtrar cuentas, editar rol/plan, abrir opciones dentro del modal, flechas, Escape primero cierra el menú, cierre exterior y tabulación. Guardar, ver confirmación y reabrir tras recargar. Los errores deben aparecer en el modal. No degradar ni suspender la cuenta del administrador que realiza la prueba.
6. Facturación: catálogo, estado, elección y sincronización; no mostrar éxito si la recarga falla. Pruebas de pagos, renovación y cancelación mediante el cliente Stripe simulado y PostgreSQL aislado. No completar cobros reales para validar.
7. Herramienta principal: éxito, error, repetir y copiar resultado. Tras un error no presentar un resultado anterior como si correspondiera a la petición nueva.
8. Mapora: búsqueda/resolución/inversa, creación y visor embebido, expiración y revocación. Vitruko: extracción, JSON, caché, ficha incompleta e imagen que falla.

## Casos de regresión en backend

Los tests protegen persistencia de perfil y cierre de sesión, permisos por cuenta, suspensión/restauración, límites de claves, cuotas concurrentes, velocidad compartida, planes ilimitados, administración sin alterar Stripe, firma/reconciliación de webhooks y uso único de tokens de recuperación. Los tests específicos de cada API cubren proveedores, normalización, caché y validación de destinos.

Conservar el borrador cuando falla una operación y mostrar el error real. Una petición correcta seguida de una recarga fallida debe comunicar el fallo de recarga; no anunciar una actualización verificada.
