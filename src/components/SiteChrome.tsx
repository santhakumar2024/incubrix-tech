import type { Booking, User } from "../types";

export type PageName = "discover" | "bookings" | "profile";

export function SiteHeader({
  user,
  bookings,
  page,
  onNavigate,
  onSignIn,
  onSignUp,
}: {
  user: User | null;
  bookings: Booking[];
  page: PageName;
  onNavigate: (page: PageName) => void;
  onSignIn: () => void;
  onSignUp: () => void;
}) {
  return (
    <header className="site-header">
      <a className="brand" href="#" onClick={() => { onNavigate("discover"); window.scrollTo({ top: 0, behavior: "smooth" }); }} aria-label="Seatflow home">
        <span className="brand-mark">s<span>.</span></span><span>seatflow</span>
      </a>
      <nav className="main-nav" aria-label="Main navigation">
        <button className={page === "discover" ? "nav-link nav-link--active" : "nav-link"} onClick={() => onNavigate("discover")}>Discover</button>
        {user && (
          <button className={page === "bookings" ? "nav-link nav-link--active" : "nav-link"} onClick={() => onNavigate("bookings")}>
            My bookings{bookings.length > 0 && <span className="nav-count">{bookings.filter((booking) => booking.status === "confirmed").length}</span>}
          </button>
        )}
        <button className={page === "profile" ? "nav-link nav-link--active" : "nav-link"} onClick={() => onNavigate("profile")}>Profile</button>
      </nav>
      <div className="header-account">
        {user ? (
          <button className="profile-shortcut" onClick={() => onNavigate("profile")} aria-label="View profile">
            <span className="user-greeting">Hey, {user.name.split(" ")[0]}</span>
            <span className="avatar">{user.name.charAt(0).toUpperCase()}</span>
          </button>
        ) : (
          <>
            <button className="sign-in-link" onClick={onSignIn}>Sign in</button>
            <button className="button button--header" onClick={onSignUp}>Join us <span aria-hidden="true">↗</span></button>
          </>
        )}
      </div>
    </header>
  );
}

export function SiteFooter({ onHome }: { onHome: () => void }) {
  return (
    <footer className="site-footer">
      <a className="brand brand--footer" href="#" onClick={onHome}><span className="brand-mark">s<span>.</span></span><span>seatflow</span></a>
      <span>Make a little room for life.</span>
      <span>MADE FOR THE MOMENTS THAT MAKE IT</span>
    </footer>
  );
}

export function Toast({ message, onDismiss }: { message: string; onDismiss: () => void }) {
  return (
    <div className="toast" role="status">
      <span>✳</span>{message}
      <button onClick={onDismiss} aria-label="Dismiss notification">×</button>
    </div>
  );
}
