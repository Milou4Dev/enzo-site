(() => {
  const e = {
      ANCHORS: 'a[href^="#"]',
      CONTACT_FORM: "#contact-form",
      NAV_LINKS: "nav a",
      CLOSE_MODAL: ".close-modal",
    },
    t = /^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/,
    n = {
      NAME_REQUIRED: "Le nom est requis",
      EMAIL_REQUIRED: "L'email est requis",
      EMAIL_INVALID: "L'email n'est pas valide",
      MESSAGE_REQUIRED: "Le message est requis",
    },
    o = (e, t) => {
      let n;
      return (...o) => {
        clearTimeout(n), (n = setTimeout(() => e.apply(null, o), t));
      };
    },
    s = (e) => {
      const t = document.createElement("div");
      return (t.textContent = e), t.innerHTML;
    },
    i = (e, t) => {
      const n = document.createElement("div");
      (n.className = "error"),
        Object.assign(n.style, {
          color: "red",
          fontSize: "0.8rem",
          marginTop: "0.25rem",
        }),
        (n.textContent = s(t)),
        e.parentNode.appendChild(n);
    },
    a = () => {
      document.querySelectorAll(".error").forEach((e) => e.remove());
    },
    r = (e) => t.test(null == e ? void 0 : e.trim()),
    l = (e, t, o) => {
      let s = !0;
      return (
        a(),
        null != (null == e ? void 0 : e.trim()) ||
          (i(document.getElementById("name"), n.NAME_REQUIRED), (s = !1)),
        null != (null == t ? void 0 : t.trim())
          ? r(t) ||
            (i(document.getElementById("email"), n.EMAIL_INVALID), (s = !1))
          : (i(document.getElementById("email"), n.EMAIL_REQUIRED), (s = !1)),
        null != (null == o ? void 0 : o.trim()) ||
          (i(document.getElementById("message"), n.MESSAGE_REQUIRED), (s = !1)),
        s
      );
    };
  class c {
    constructor(t, n) {
      (this.modal = document.getElementById(t)),
        this.modal &&
          ((this.closeBtn = this.modal.querySelector(e.CLOSE_MODAL)),
          this.setupEventListeners(n));
    }
    setupEventListeners(e) {
      document.querySelectorAll(e).forEach((e) => {
        e.addEventListener("click", this.open.bind(this));
      }),
        this.closeBtn &&
          this.closeBtn.addEventListener("click", this.close.bind(this)),
        this.modal.addEventListener("click", (e) => {
          e.target === this.modal && this.close();
        }),
        document.addEventListener("keydown", (e) => {
          "Escape" === e.key &&
            this.modal.classList.contains("active") &&
            this.close();
        });
    }
    open(e) {
      null == e || e.preventDefault(), this.modal.classList.add("active");
    }
    close() {
      this.modal.classList.remove("active");
    }
  }
  const d = () => {
      const t = o((e) => {
        e.scrollIntoView({ behavior: "smooth", block: "start" });
      }, 50);
      document.querySelectorAll(e.ANCHORS).forEach((n) => {
        n.addEventListener("click", (o) => {
          o.preventDefault();
          const s = n.getAttribute("href").substring(1);
          if (!s) return;
          const i = document.getElementById(s);
          i &&
            (t(i),
            document
              .querySelectorAll(e.NAV_LINKS)
              .forEach((e) => e.classList.remove("active")),
            n.classList.add("active"));
        });
      });
    },
    m = () => {
      const t = document.querySelector(e.CONTACT_FORM);
      t &&
        t.addEventListener("submit", (e) => {
          e.preventDefault();
          const n = new FormData(t),
            o = n.get("name"),
            s = n.get("email"),
            i = n.get("message");
          if (!l(o, s, i)) return;
          const a = t.querySelector('button[type="submit"]');
          (a.disabled = !0),
            (a.textContent = "Envoi en cours..."),
            fetch(t.action, {
              method: "POST",
              body: n,
              headers: { Accept: "application/json" },
            })
              .then((e) => {
                if (!e.ok) throw new Error("Network response was not ok");
                t.reset(), alert("Message envoyé avec succès!");
              })
              .catch((e) => {
                console.error("Error:", e),
                  alert(
                    "Une erreur est survenue. Veuillez réessayer plus tard."
                  );
              })
              .finally(() => {
                (a.disabled = !1), (a.textContent = "Envoyer le message");
              });
        });
    };
  document.addEventListener("DOMContentLoaded", () => {
    try {
      d(),
        m(),
        new c("development-modal", ".read-more"),
        new c("legal-modal", ".mentions-legales");
    } catch (e) {
      console.error("Initialization error:", e);
    }
  });
})();
