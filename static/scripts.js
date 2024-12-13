document.querySelectorAll('a[href^="#"]').forEach((anchor) => {
  anchor.addEventListener("click", function (e) {
    e.preventDefault();
    const t = this.getAttribute("href").substring(1);
    if (!t) return;
    const n = document.getElementById(t);
    n &&
      (n.scrollIntoView({ behavior: "smooth" }),
      document
        .querySelectorAll("nav a")
        .forEach((e) => e.classList.remove("active")),
      this.classList.add("active"));
  });
});
const contactForm = document.getElementById("contact-form");
function showError(e, t) {
  const n = document.createElement("div");
  (n.className = "error"),
    (n.style.color = "red"),
    (n.style.fontSize = "0.8rem"),
    (n.style.marginTop = "0.25rem"),
    (n.textContent = t),
    e.parentNode.appendChild(n);
}
function isValidEmail(e) {
  return /^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(e);
}
function setupModal(e, t) {
  const n = document.getElementById(e);
  if (!n) return;
  const o = n.querySelector(".close-modal");
  o &&
    (document.querySelectorAll(t).forEach((e) => {
      e.addEventListener("click", function (e) {
        e.preventDefault(), n.classList.add("active");
      });
    }),
    o.addEventListener("click", () => {
      n.classList.remove("active");
    }),
    n.addEventListener("click", (e) => {
      e.target === n && n.classList.remove("active");
    }));
}
contactForm &&
  contactForm.addEventListener("submit", function (e) {
    e.preventDefault();
    const t = this.querySelector("#name").value.trim(),
      n = this.querySelector("#email").value.trim(),
      o = this.querySelector("#message").value.trim();
    document.querySelectorAll(".error").forEach((e) => e.remove());
    let r = !1;
    t ||
      (showError(this.querySelector("#name"), "Le nom est requis"), (r = !0)),
      n
        ? isValidEmail(n) ||
          (showError(this.querySelector("#email"), "L'email n'est pas valide"),
          (r = !0))
        : (showError(this.querySelector("#email"), "L'email est requis"),
          (r = !0)),
      o ||
        (showError(this.querySelector("#message"), "Le message est requis"),
        (r = !0)),
      r || this.submit();
  }),
  setupModal("development-modal", ".read-more"),
  setupModal("legal-modal", ".mentions-legales");
