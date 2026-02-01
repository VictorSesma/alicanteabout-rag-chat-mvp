(function () {
  const config = window.AATChatWidgetConfig || {};
  const buttonLabel = config.buttonLabel || "Ask Alicante";
  const accentColor = config.accentColor || "#0f4c5c";
  const allowedLanguages = (config.allowedLanguages || "").split(",").map((v) => v.trim()).filter(Boolean);
  const buttonPosition = (config.buttonPosition || "bottom-right").toLowerCase();
  const offsetX = Number.isFinite(Number(config.offsetX)) ? Number(config.offsetX) : 20;
  const offsetY = Number.isFinite(Number(config.offsetY)) ? Number(config.offsetY) : 20;

  function createButton() {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "aat-chat-button";
    button.textContent = buttonLabel;
    button.style.setProperty("--aat-accent", accentColor);
    applyPosition(button);
    button.addEventListener("click", () => {
      if (window.AATChatWidget && window.AATChatWidget.open) {
        window.AATChatWidget.open();
        return;
      }
      loadWidgetAssets(() => {
        if (window.AATChatWidget && window.AATChatWidget.open) {
          window.AATChatWidget.open();
        }
      });
    });
    document.body.appendChild(button);
  }

  function loadWidgetAssets(onReady) {
    const existing = document.getElementById("aat-chat-widget-css");
    if (!existing) {
      const link = document.createElement("link");
      link.id = "aat-chat-widget-css";
      link.rel = "stylesheet";
      link.href = config.cssUrl || "";
      link.type = "text/css";
      document.head.appendChild(link);
    }
    const script = document.createElement("script");
    script.src = config.jsUrl || "";
    script.onload = onReady;
    document.body.appendChild(script);
  }

  function injectBaseStyles() {
    const style = document.createElement("style");
    style.textContent = `
      .aat-chat-button {
        position: fixed;
        background: var(--aat-accent, ${accentColor});
        color: #fff;
        border: 0;
        padding: 12px 16px;
        border-radius: 999px;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        box-shadow: 0 10px 24px rgba(0,0,0,0.2);
        z-index: 9999;
      }
      .aat-chat-button:hover { opacity: 0.92; }
    `;
    document.head.appendChild(style);
  }

  function applyPosition(button) {
    const x = Math.max(0, offsetX);
    const y = Math.max(0, offsetY);
    button.style.left = "";
    button.style.right = "";
    button.style.top = "";
    button.style.bottom = "";
    button.style.transform = "";
    switch (buttonPosition) {
      case "bottom-left":
        button.style.left = x + "px";
        button.style.bottom = y + "px";
        break;
      case "top-right":
        button.style.right = x + "px";
        button.style.top = y + "px";
        break;
      case "top-left":
        button.style.left = x + "px";
        button.style.top = y + "px";
        break;
      case "bottom-right":
      default:
        button.style.right = x + "px";
        button.style.bottom = y + "px";
        break;
    }
  }

  function init() {
    if (!config.jsUrl || !config.cssUrl) {
      console.warn("AAT Chat Widget missing asset URLs.");
    }
    if (!isLanguageAllowed()) {
      return;
    }
    injectBaseStyles();
    createButton();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  function isLanguageAllowed() {
    if (!allowedLanguages.length) {
      return true;
    }
    const lang = (document.documentElement.getAttribute("lang") || "").toLowerCase();
    if (!lang) {
      return false;
    }
    return allowedLanguages.some(
      (code) => lang === code.toLowerCase() || lang.startsWith(code.toLowerCase() + "-")
    );
  }
})();
