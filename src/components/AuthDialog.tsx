import { useState, type FormEvent } from "react";
import { api } from "../api";
import type { User } from "../types";

export type AuthMode = "login" | "register";

export default function AuthDialog({
  mode,
  onClose,
  onModeChange,
  onSuccess,
}: {
  mode: AuthMode;
  onClose: () => void;
  onModeChange: (mode: AuthMode) => void;
  onSuccess: (user: User) => void;
}) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      const user = mode === "register"
        ? await api.signUp(name, email, password)
        : await api.signIn(email, password);
      onSuccess(user);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not sign you in.");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="modal auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-title">
        <button className="modal-close" onClick={onClose} aria-label="Close dialog">×</button>
        <span className="modal-kicker">GOOD THINGS AWAIT</span>
        <h2 id="auth-title">{mode === "register" ? "Get in on it." : "Welcome back."}</h2>
        <p className="modal-subtitle">
          {mode === "register" ? "Create an account to save your seat." : "Your next favorite night is just a sign in away."}
        </p>
        <form className="form-stack" onSubmit={submit}>
          {mode === "register" && (
            <label>
              Your name
              <input autoComplete="name" value={name} onChange={(event) => setName(event.target.value)} required maxLength={80} />
            </label>
          )}
          <label>
            Email address
            <input type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" autoComplete={mode === "register" ? "new-password" : "current-password"} value={password} onChange={(event) => setPassword(event.target.value)} required minLength={8} maxLength={72} />
            {mode === "register" && <span className="field-note">At least 8 characters</span>}
          </label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="button button--wide" disabled={pending}>
            {pending ? "One moment..." : mode === "register" ? "Create your account" : "Sign in"}
          </button>
        </form>
        <p className="auth-switch">
          {mode === "register" ? "Already have an account?" : "New around here?"}{" "}
          <button onClick={() => onModeChange(mode === "register" ? "login" : "register")}>
            {mode === "register" ? "Sign in" : "Create an account"}
          </button>
        </p>
      </section>
    </div>
  );
}
