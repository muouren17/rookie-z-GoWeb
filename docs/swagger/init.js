window.addEventListener("load", async function () {
  try {
    const response = await fetch("./swagger.json", { cache: "no-store" });
    if (!response.ok) throw new Error("HTTP " + response.status);
    const specification = await response.json();
    // Only the displayed copy uses the existing documentation proxy.
    // The generated file retains the backend host from main.go annotations.
    specification.host = window.location.host;
    specification.schemes = [window.location.protocol.replace(":", "")];
    window.ui = SwaggerUIBundle({
    spec: specification,
    dom_id: "#swagger-ui",
    deepLinking: true,
    filter: true,
    docExpansion: "list",
    defaultModelsExpandDepth: 1,
    displayRequestDuration: true,
    persistAuthorization: false,
    tryItOutEnabled: false,
    supportedSubmitMethods: ["get", "post"],
    validatorUrl: null,
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout"
    });
  } catch (error) {
    document.getElementById("swagger-ui").textContent =
      "无法加载 swagger.json。请在项目根目录执行 sh docs/swagger/generate.sh，再刷新页面。";
    console.error("Swagger document loading failed:", error.message);
  }
});
