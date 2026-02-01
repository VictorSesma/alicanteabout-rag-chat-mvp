(function () {
  const config = window.AATChatWidgetConfig || {};
  const apiUrl = config.apiUrl || "https://api.alicanteabout.com/chat";
  const tokenUrl = config.tokenUrl || "/wp-json/alicanteabout/v1/chat-token";
  const accentColor = config.accentColor || "#0f4c5c";
  const title = config.title || "Ask Alicante";
  const disclaimer = config.disclaimer || "Answers are based on AlicanteAbout content.";
  const gdprDisclaimer =
    config.gdprDisclaimer ||
    "We log questions for quality. Please avoid personal data.";

  let modal;
  let overlay;
  let messagesEl;
  let inputEl;
  let sendBtn;
  let tokenCache = null;
  let tokenExpiresAt = 0;

  function buildModal() {
    overlay = document.createElement("div");
    overlay.className = "aat-chat-overlay";
    overlay.addEventListener("click", close);

    modal = document.createElement("div");
    modal.className = "aat-chat-modal";
    modal.style.setProperty("--aat-accent", accentColor);

    const header = document.createElement("div");
    header.className = "aat-chat-header";
    header.innerHTML = `
      <div class="aat-chat-title">${escapeHtml(title)}</div>
      <button type="button" class="aat-chat-close" aria-label="Close">×</button>
    `;
    header.querySelector(".aat-chat-close").addEventListener("click", close);

    messagesEl = document.createElement("div");
    messagesEl.className = "aat-chat-messages";

    const footer = document.createElement("div");
    footer.className = "aat-chat-footer";

    inputEl = document.createElement("textarea");
    inputEl.className = "aat-chat-input";
    inputEl.rows = 2;
    inputEl.placeholder = "Ask a question about Alicante...";
    inputEl.addEventListener("keydown", onInputKey);

    sendBtn = document.createElement("button");
    sendBtn.type = "button";
    sendBtn.className = "aat-chat-send";
    sendBtn.textContent = "Send";
    sendBtn.addEventListener("click", sendMessage);

    const disclaimers = document.createElement("div");
    disclaimers.className = "aat-chat-disclaimer";
    disclaimers.innerHTML = `
      <div>${escapeHtml(disclaimer)}</div>
      <div>${escapeHtml(gdprDisclaimer)}</div>
    `;

    footer.appendChild(inputEl);
    footer.appendChild(sendBtn);

    modal.appendChild(header);
    modal.appendChild(messagesEl);
    modal.appendChild(footer);
    modal.appendChild(disclaimers);
  }

  function open() {
    if (!modal) {
      buildModal();
    }
    document.body.appendChild(overlay);
    document.body.appendChild(modal);
    document.body.classList.add("aat-chat-open");
    inputEl.focus();
  }

  function close() {
    if (modal && modal.parentNode) {
      modal.parentNode.removeChild(modal);
    }
    if (overlay && overlay.parentNode) {
      overlay.parentNode.removeChild(overlay);
    }
    document.body.classList.remove("aat-chat-open");
    resetConversation();
  }

  function resetConversation() {
    if (messagesEl) {
      messagesEl.innerHTML = "";
    }
    tokenCache = null;
    tokenExpiresAt = 0;
  }

  function onInputKey(e) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  function appendMessage(role, text) {
    const msg = document.createElement("div");
    msg.className = "aat-chat-message aat-" + role;
    const textEl = document.createElement("div");
    textEl.className = "aat-chat-message-text";
    textEl.textContent = text || "";
    msg.appendChild(textEl);
    if (role === "assistant") {
      const sourcesEl = document.createElement("div");
      sourcesEl.className = "aat-chat-sources";
      msg.appendChild(sourcesEl);
      const typingEl = document.createElement("div");
      typingEl.className = "aat-chat-typing";
      typingEl.innerHTML = "<span></span><span></span><span></span>";
      msg.appendChild(typingEl);
    }
    messagesEl.appendChild(msg);
    messagesEl.scrollTop = messagesEl.scrollHeight;
    return msg;
  }

  async function getToken() {
    const now = Date.now();
    if (tokenCache && tokenExpiresAt - now > 10_000) {
      return tokenCache;
    }
    const res = await fetch(tokenUrl, { credentials: "same-origin" });
    if (!res.ok) {
      throw new Error("token fetch failed");
    }
    const data = await res.json();
    tokenCache = data.token;
    tokenExpiresAt = now + (data.expires_in || 120) * 1000;
    return tokenCache;
  }

  async function sendMessage() {
    const text = inputEl.value.trim();
    if (!text) {
      return;
    }
    inputEl.value = "";
    const userMsg = appendMessage("user", text);
    userMsg.scrollIntoView({ block: "end" });

    let assistantMsg = appendMessage("assistant", "");
    setTyping(assistantMsg, true);
    try {
      const token = await getToken();
      await streamAnswer(text, token, assistantMsg);
    } catch (err) {
      setTyping(assistantMsg, false);
      setMessageText(
        assistantMsg,
        "Sorry, I couldn't get an answer right now. Please try again."
      );
    }
  }

  async function streamAnswer(question, token, assistantMsg, retry) {
    try {
      const res = await fetch(apiUrl + "?stream=1", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
          Authorization: "Bearer " + token,
        },
        body: JSON.stringify({ question, lang: "en" }),
      });
      if (res.status === 401 && !retry) {
        tokenCache = null;
        tokenExpiresAt = 0;
        const fresh = await getToken();
        return streamAnswer(question, fresh, assistantMsg, true);
      }
      if (!res.ok || !res.body) {
        throw new Error("chat request failed");
      }

      const contentType = (res.headers.get("Content-Type") || "").toLowerCase();
      if (!contentType.includes("text/event-stream")) {
        const payload = await res.json();
        if (payload && payload.answer) {
          setMessageText(assistantMsg, payload.answer);
        }
        if (payload && payload.sources) {
          setSources(assistantMsg, payload.sources);
        }
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      const state = { buffer: "", started: false };
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split("\n\n");
        buffer = parts.pop() || "";
        parts.forEach((chunk) => handleSseChunk(chunk, assistantMsg, state));
      }
    } finally {
      setTyping(assistantMsg, false);
    }
  }

  function handleSseChunk(chunk, assistantMsg, state) {
    const lines = chunk.split("\n");
    let event = "message";
    let data = "";
    lines.forEach((line) => {
      if (line.startsWith("event:")) {
        event = line.replace("event:", "").trim();
      } else if (line.startsWith("data:")) {
        data += line.replace("data:", "").trim();
      }
    });
    if (!data) {
      return;
    }
    if (!state.started) {
      state.started = true;
      setTyping(assistantMsg, false);
    }
    if (event === "delta") {
      let delta = data;
      try {
        const payload = JSON.parse(data);
        if (payload && typeof payload.delta === "string") {
          delta = payload.delta;
        }
      } catch (err) {
        delta = data;
      }
      state.buffer += delta;
      const answer = extractAnswerFromBuffer(state.buffer);
      if (answer !== null) {
        setMessageText(assistantMsg, answer);
      }
    } else if (event === "result") {
      try {
        const payload = JSON.parse(data);
        if (payload.answer) {
          setMessageText(assistantMsg, payload.answer);
        }
        if (payload.sources) {
          setSources(assistantMsg, payload.sources);
        }
      } catch (err) {
        return;
      }
    } else if (event === "error") {
      setMessageText(
        assistantMsg,
        "Sorry, I couldn't get an answer right now. Please try again."
      );
    }
  }

  function setMessageText(msg, text) {
    const el = msg.querySelector(".aat-chat-message-text");
    if (el) {
      el.textContent = text;
    } else {
      msg.textContent = text;
    }
  }

  function setTyping(msg, active) {
    if (active) {
      msg.classList.add("aat-typing");
    } else {
      msg.classList.remove("aat-typing");
    }
  }

  function setSources(msg, sources) {
    const el = msg.querySelector(".aat-chat-sources");
    if (!el) {
      return;
    }
    el.innerHTML = "";
    if (!Array.isArray(sources) || sources.length === 0) {
      return;
    }
    const title = document.createElement("div");
    title.className = "aat-chat-sources-title";
    title.textContent = "Related articles";
    const list = document.createElement("ul");
    list.className = "aat-chat-sources-list";
    sources.forEach((src) => {
      const li = document.createElement("li");
      const link = document.createElement("a");
      link.href = src.url;
      link.target = "_blank";
      link.rel = "noopener";
      link.textContent = src.title || src.url || "Source";
      li.appendChild(link);
      list.appendChild(li);
    });
    el.appendChild(title);
    el.appendChild(list);
  }

  function extractAnswerFromBuffer(buffer) {
    try {
      const parsed = JSON.parse(buffer);
      if (parsed && typeof parsed.answer === "string") {
        return parsed.answer;
      }
    } catch (err) {
      // fall through to partial extraction
    }
    const keyIndex = buffer.indexOf('"answer"');
    if (keyIndex === -1) {
      return null;
    }
    let i = buffer.indexOf(":", keyIndex);
    if (i === -1) {
      return null;
    }
    i += 1;
    while (i < buffer.length && /\s/.test(buffer[i])) {
      i += 1;
    }
    if (buffer[i] !== '"') {
      return null;
    }
    i += 1;
    let out = "";
    let escaped = false;
    for (; i < buffer.length; i += 1) {
      const ch = buffer[i];
      if (escaped) {
        switch (ch) {
          case "n":
            out += "\n";
            break;
          case "t":
            out += "\t";
            break;
          case "r":
            out += "\r";
            break;
          case '"':
            out += '"';
            break;
          case "\\":
            out += "\\";
            break;
          case "u": {
            const hex = buffer.slice(i + 1, i + 5);
            if (/^[0-9a-fA-F]{4}$/.test(hex)) {
              out += String.fromCharCode(parseInt(hex, 16));
              i += 4;
            } else {
              out += "u";
            }
            break;
          }
          default:
            out += ch;
            break;
        }
        escaped = false;
        continue;
      }
      if (ch === "\\") {
        escaped = true;
        continue;
      }
      if (ch === '"') {
        return out;
      }
      out += ch;
    }
    return out;
  }

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  function init() {
    window.AATChatWidget = { open, close };
  }

  init();
})();
