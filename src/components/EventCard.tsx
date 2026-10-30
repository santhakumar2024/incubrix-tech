import type { EventListing } from "../types";
import { CalendarIcon, formatDate, formatPrice, formatTime, PinIcon, SparkIcon } from "./Icons";

function EventArtwork({ accent, category }: { accent: string; category: string }) {
  return (
    <div className={`event-art event-art--${accent}`} aria-label={`${category} event artwork`} role="img">
      <div className="art-orbit art-orbit--one" />
      <div className="art-orbit art-orbit--two" />
      <div className="art-disc"><SparkIcon /></div>
      <span className="art-caption">{category}</span>
      <span className="art-mark">S<span>.</span></span>
    </div>
  );
}

export default function EventCard({
  event,
  onBook,
}: {
  event: EventListing;
  onBook: (event: EventListing) => void;
}) {
  const soldOut = event.available <= 0;
  return (
    <article className="event-card">
      <EventArtwork accent={event.accent} category={event.category} />
      <div className="event-card__body">
        <div className="event-card__eyebrow">
          <span>{event.category}</span>
          <span className={event.available <= 8 && !soldOut ? "availability availability--low" : "availability"}>
            {soldOut ? "Sold out" : `${event.available} spots left`}
          </span>
        </div>
        <h3>{event.title}</h3>
        <p className="event-description">{event.description}</p>
        <div className="event-meta">
          <span><CalendarIcon />{formatDate(event.startsAt)} · {formatTime(event.startsAt)}</span>
          <span><PinIcon />{event.venue}, {event.city}</span>
        </div>
        <div className="event-card__footer">
          <p><strong>{formatPrice(event.priceCents)}</strong><span> / person</span></p>
          <button className="button button--small" disabled={soldOut} onClick={() => onBook(event)}>
            {soldOut ? "Sold out" : "Find your spot"} <span aria-hidden="true">↗</span>
          </button>
        </div>
      </div>
    </article>
  );
}
