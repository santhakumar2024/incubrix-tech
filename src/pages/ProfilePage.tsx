import type { Booking, User } from "../types";

export default function ProfilePage({
  user,
  bookings,
  error,
  onSignIn,
  onSignUp,
  onSignOut,
  onBookings,
  onDiscover,
}: {
  user: User | null;
  bookings: Booking[];
  error: string;
  onSignIn: () => void;
  onSignUp: () => void;
  onSignOut: () => void;
  onBookings: () => void;
  onDiscover: () => void;
}) {
  const confirmedBookings = bookings.filter((booking) => booking.status === "confirmed");
  const upcomingBookings = confirmedBookings.filter((booking) => new Date(booking.startsAt) > new Date());

  if (!user) {
    return (
      <section className="profile-page">
        <div className="profile-heading">
          <span className="section-kicker">YOUR SEATFLOW ACCOUNT</span>
          <h1>Your plans.<br /><em>Your place.</em></h1>
          <p>Sign in to keep your bookings together, pick your seats, and make the good times yours.</p>
        </div>
        {error && <div className="error-banner" role="alert">{error}</div>}
        <div className="profile-guest-card">
          <div className="profile-guest-mark">s<span>.</span></div>
          <div>
            <h2>Make yourself at home.</h2>
            <p>Your account keeps your event plans and seat bookings in one easy place.</p>
          </div>
          <div className="profile-actions">
            <button className="button" onClick={onSignIn}>Sign in</button>
            <button className="button button--ghost" onClick={onSignUp}>Create an account</button>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="profile-page">
      <div className="profile-heading">
        <span className="section-kicker">YOUR SEATFLOW ACCOUNT</span>
        <h1>Good to see you,<br /><em>{user.name.split(" ")[0]}.</em></h1>
        <p>Your details and plans, all in one place.</p>
      </div>

      {error && <div className="error-banner" role="alert">{error}</div>}

      <div className="profile-layout">
        <section className="profile-card" aria-labelledby="profile-details-title">
          <div className="profile-card__heading">
            <div className="profile-avatar">{user.name.charAt(0).toUpperCase()}</div>
            <div><span className="section-kicker">YOUR ACCOUNT</span><h2 id="profile-details-title">The basics</h2></div>
          </div>
          <dl className="profile-details">
            <div><dt>Full name</dt><dd>{user.name}</dd></div>
            <div><dt>Email address</dt><dd>{user.email}</dd></div>
            <div><dt>Account number</dt><dd>SF-{String(user.id).padStart(6, "0")}</dd></div>
          </dl>
          <button className="button button--ghost profile-signout" onClick={onSignOut}>Sign out <span aria-hidden="true">↗</span></button>
        </section>

        <section className="profile-activity" aria-labelledby="profile-activity-title">
          <div className="profile-card__heading">
            <div><span className="section-kicker">OUT THERE, LIVING IT</span><h2 id="profile-activity-title">Your plans</h2></div>
            <button className="profile-text-link" onClick={onBookings}>See bookings ↗</button>
          </div>
          <div className="profile-stats">
            <div><strong>{upcomingBookings.length}</strong><span>upcoming</span></div>
            <div><strong>{confirmedBookings.length}</strong><span>confirmed</span></div>
            <div><strong>{bookings.filter((booking) => booking.status === "cancelled").length}</strong><span>cancelled</span></div>
          </div>
          <div className="profile-plan-note">
            <span className="profile-plan-note__icon">✳</span>
            <div><strong>There's always something good on.</strong><p>Find your next favorite thing and pick your perfect spot.</p></div>
          </div>
          <button className="button" onClick={onDiscover}>Explore events <span aria-hidden="true">↗</span></button>
        </section>
      </div>
    </section>
  );
}
