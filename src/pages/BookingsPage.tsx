import type { Booking } from "../types";
import { CalendarIcon, formatPrice, formatTime } from "../components/Icons";

function BookingList({
  bookings,
  onCancel,
  onDiscover,
}: {
  bookings: Booking[];
  onCancel: (booking: Booking) => void;
  onDiscover: () => void;
}) {
  if (bookings.length === 0) {
    return (
      <section className="empty-state">
        <div className="empty-icon"><CalendarIcon /></div>
        <span className="section-kicker">YOUR NEXT STORY STARTS HERE</span>
        <h2>No plans on the calendar. Yet.</h2>
        <p>Find something that feels like you and make a night of it.</p>
        <button className="button" onClick={onDiscover}>Explore what's on</button>
      </section>
    );
  }

  return (
    <section className="booking-list" aria-label="Your bookings">
      {bookings.map((booking) => (
        <article className={`booking-row ${booking.status === "cancelled" ? "booking-row--cancelled" : ""}`} key={booking.id}>
          <div className={`booking-date booking-date--${booking.status === "cancelled" ? "muted" : "active"}`}>
            <span>{new Intl.DateTimeFormat("en-US", { month: "short" }).format(new Date(booking.startsAt))}</span>
            <strong>{new Intl.DateTimeFormat("en-US", { day: "2-digit" }).format(new Date(booking.startsAt))}</strong>
          </div>
          <div className="booking-info">
            <div className="event-card__eyebrow">
              <span>{booking.category}</span>
              <span className={`status-pill status-pill--${booking.status}`}>{booking.status}</span>
            </div>
            <h3>{booking.title}</h3>
            <p>{booking.venue}, {booking.city} · {formatTime(booking.startsAt)}</p>
            {booking.seatLabels.length > 0 && <span className="booking-seats">Seats {booking.seatLabels.join(", ")}</span>}
            <span className="booking-count">{booking.quantity} {booking.quantity === 1 ? "seat" : "seats"} · {formatPrice(booking.priceCents * booking.quantity)}</span>
          </div>
          {booking.status === "confirmed" && new Date(booking.startsAt) > new Date() && (
            <button className="button button--ghost" onClick={() => onCancel(booking)}>Cancel booking</button>
          )}
        </article>
      ))}
    </section>
  );
}

export default function BookingsPage({
  bookings,
  error,
  onRetry,
  onCancel,
  onDiscover,
}: {
  bookings: Booking[];
  error: string;
  onRetry: () => void;
  onCancel: (booking: Booking) => void;
  onDiscover: () => void;
}) {
  return (
    <section className="bookings-page">
      <div className="bookings-heading">
        <div>
          <span className="section-kicker">YOUR PLANS, ALL IN ONE PLACE</span>
          <h1>Your <em>good times.</em></h1>
          <p>Everything you have coming up, right here.</p>
        </div>
        <button className="button button--ghost" onClick={onDiscover}>＋ Find another plan</button>
      </div>
      {error && <div className="error-banner" role="alert"><span>{error}</span><button onClick={onRetry}>Try again</button></div>}
      <BookingList bookings={bookings} onCancel={onCancel} onDiscover={onDiscover} />
    </section>
  );
}
