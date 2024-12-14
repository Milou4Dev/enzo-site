(() => {
    const SELECTORS = {
        ANCHORS: 'a[href^="#"]', CONTACT_FORM: "#contact-form", NAV_LINKS: "nav a", CLOSE_MODAL: ".close-modal",
    };

    const EMAIL_REGEX = /^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;

    const ERROR_MESSAGES = {
        NAME_REQUIRED: "Le nom est requis",
        EMAIL_REQUIRED: "L'email est requis",
        EMAIL_INVALID: "L'email n'est pas valide",
        MESSAGE_REQUIRED: "Le message est requis",
    };

    const debounce = (callback, delay) => {
        let timeoutId;
        return (...args) => {
            clearTimeout(timeoutId);
            timeoutId = setTimeout(() => callback.apply(null, args), delay);
        };
    };

    const escapeHtml = (text) => {
        const div = document.createElement("div");
        div.textContent = text;
        return div.innerHTML;
    };

    const showError = (element, message) => {
        const errorDiv = document.createElement("div");
        errorDiv.className = "error";
        Object.assign(errorDiv.style, {
            color: "red", fontSize: "0.8rem", marginTop: "0.25rem",
        });
        errorDiv.textContent = escapeHtml(message);
        element.parentNode.appendChild(errorDiv);
    };

    const clearErrors = () => {
        document.querySelectorAll(".error").forEach((error) => error.remove());
    };

    const isValidEmail = (email) => EMAIL_REGEX.test(email?.trim());

    const validateForm = (name, email, message) => {
        let isValid = true;
        clearErrors();

        if (!(name?.trim())) {
            showError(document.getElementById("name"), ERROR_MESSAGES.NAME_REQUIRED);
            isValid = false;
        }

        if (!email?.trim()) {
            showError(document.getElementById("email"), ERROR_MESSAGES.EMAIL_REQUIRED);
            isValid = false;
        } else if (!isValidEmail(email)) {
            showError(document.getElementById("email"), ERROR_MESSAGES.EMAIL_INVALID);
            isValid = false;
        }

        if (!(message?.trim())) {
            showError(document.getElementById("message"), ERROR_MESSAGES.MESSAGE_REQUIRED);
            isValid = false;
        }

        return isValid;
    };

    class Modal {
        constructor(modalId, triggerSelector) {
            this.modal = document.getElementById(modalId);
            if (this.modal) {
                this.closeBtn = this.modal.querySelector(SELECTORS.CLOSE_MODAL);
                this.setupEventListeners(triggerSelector);
            }
        }

        setupEventListeners(triggerSelector) {
            document.querySelectorAll(triggerSelector).forEach((trigger) => {
                trigger.addEventListener("click", this.open.bind(this));
            });

            if (this.closeBtn) {
                this.closeBtn.addEventListener("click", this.close.bind(this));
            }

            this.modal.addEventListener("click", (event) => {
                if (event.target === this.modal) {
                    this.close();
                }
            });

            document.addEventListener("keydown", (event) => {
                if (event.key === "Escape" && this.modal.classList.contains("active")) {
                    this.close();
                }
            });
        }

        open(event) {
            event?.preventDefault();
            this.modal.classList.add("active");
        }

        close() {
            this.modal.classList.remove("active");
        }
    }

    const setupSmoothScroll = () => {
        const scrollToElement = debounce((element) => {
            element.scrollIntoView({behavior: "smooth", block: "start"});
        }, 50);

        document.querySelectorAll(SELECTORS.ANCHORS).forEach((anchor) => {
            anchor.addEventListener("click", (event) => {
                event.preventDefault();
                const targetId = anchor.getAttribute("href").substring(1);
                if (!targetId) return;

                const targetElement = document.getElementById(targetId);
                if (targetElement) {
                    scrollToElement(targetElement);
                    document
                        .querySelectorAll(SELECTORS.NAV_LINKS)
                        .forEach((link) => link.classList.remove("active"));
                    anchor.classList.add("active");
                }
            });
        });
    };

    const setupContactForm = () => {
        const form = document.querySelector(SELECTORS.CONTACT_FORM);
        if (!form) return;

        form.addEventListener("submit", (event) => {
            event.preventDefault();
            const formData = new FormData(form);
            const name = formData.get("name");
            const email = formData.get("email");
            const message = formData.get("message");

            if (!validateForm(name, email, message)) return;

            const submitButton = form.querySelector('button[type="submit"]');
            submitButton.disabled = true;
            submitButton.textContent = "Envoi en cours...";

            fetch(form.action, {
                method: "POST", body: formData, headers: {Accept: "application/json"},
            })
                .then((response) => {
                    if (!response.ok) throw new Error("Network response was not ok");
                    form.reset();
                    alert("Message envoyé avec succès!");
                })
                .catch((error) => {
                    console.error("Error:", error);
                    alert("Une erreur est survenue. Veuillez réessayer plus tard.");
                })
                .finally(() => {
                    submitButton.disabled = false;
                    submitButton.textContent = "Envoyer le message";
                });
        });
    };

    document.addEventListener("DOMContentLoaded", () => {
        try {
            setupSmoothScroll();
            setupContactForm();
            new Modal("development-modal", ".read-more");
            new Modal("legal-modal", ".mentions-legales");
        } catch (error) {
            console.error("Initialization error:", error);
        }
    });
})();
