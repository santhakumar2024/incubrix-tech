import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "../api";
import type { EventListing, EventSeat } from "../types";
import { formatDate, formatPrice, formatTime } from "./Icons";

export default function BookingDialog({
  event,
  onClose,
  onBooked,
}: {
  event: EventListing;
  onClose: () => void;
  onBooked: () => void;
}) {
  const [seats, setSeats] = useState<EventSeat[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const selectedRef = useRef<string[]>([]);

  function updateSelected(next: string[]) {
    selectedRef.current = next;
    setSelected(next);
  }

  const loadSeats = useCallback(async () => {
    try {
      const currentSeats = await api.seats(event.id);
      setSeats(currentSeats);
      const available = new Set(currentSeats.filter((seat) => seat.available).map((seat) => seat.label));
      const previous = selectedRef.current;
      const next = previous.filter((label) => available.has(label));
      updateSelected(next);
      if (next.length !== previous.length) {
        setError("A seat you selected was just booked. Please review your seats.");
      } else {
        setError((current) => current.startsWith("Could not load") ? "" : current);
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not load the seat map.");
    } finally {
      setLoading(false);
    }
  }, [event.id]);

  useEffect(() => {
    void loadSeats();
    const stream = new EventSource("/api/availability/stream", { withCredentials: true });
    stream.addEventListener("availability", () => { void loadSeats(); });
    const refresh = window.setInterval(() => { void loadSeats(); }, 15000);
    return () => {
      stream.close();
      window.clearInterval(refresh);
    };
  }, [loadSeats]);

  const rows = useMemo(() => {
    const grouped = new Map<string, EventSeat[]>();
    for (const seat of seats) {
      const row = grouped.get(seat.row) ?? [];
      row.push(seat);
      grouped.set(seat.row, row);
    }
    return Array.from(grouped.entries());
  }, [seats]);

  function toggleSeat(seat: EventSeat) {
    if (!seat.available) return;
    setError("");
    if (selected.includes(seat.label)) {
      updateSelected(selected.filter((label) => label !== seat.label));
    } else if (selected.length >= 10) {
      setError("You can choose up to 10 seats per booking.");
    } else {
      updateSelected([...selected, seat.label]);
    }
  }

  async function submit() {
    if (selected.length === 0) {
      setError("Choose at least one seat to continue.");
      return;
    }
    setPending(true);
    setError("");
    try {
      await api.book(event.id, selected);
      onBooked();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not complete your booking.");
      await loadSeats();
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(mouseEvent) => mouseEvent.target === mouseEvent.currentTarget && onClose()}>
      <section className="modal booking-modal seat-modal" role="dialog" aria-modal="true" aria-labelledby="booking-title">
        <button className="modal-close" onClick={onClose} aria-label="Close dialog">×</button>
        <span className="modal-kicker">PICK YOUR PERFECT SPOT</span>
        <h2 id="booking-title">{event.title}</h2>
        <p className="modal-subtitle">{formatDate(event.startsAt)} at {formatTime(event.startsAt)} · {event.venue}</p>

        <div className="seat-screen"><span>SCREEN / STAGE</span></div>

        {loading ? (
          <div className="seat-loading"><span className="spinner" /> Loading available seats...</div>
        ) : seats.length === 0 ? (
          <div className="seat-loading">No seat map is available for this event.</div>
        ) : (
          <div className="seat-map" aria-label="Choose your seats">
            {rows.map(([rowLabel, rowSeats]) => (
              <div className="seat-row" key={rowLabel}>
                <span className="seat-row-label">{rowLabel}</span>
                <div className="seat-row__chairs">
                  {rowSeats.map((seat) => {
                    const isSelected = selected.includes(seat.label);
                    return (
                      <button
                        key={seat.label}
                        className={`seat ${isSelected ? "seat--selected" : ""} ${!seat.available ? "seat--unavailable" : ""}`}
                        aria-label={`Row ${seat.row}, seat ${seat.number}${!seat.available ? ", booked" : isSelected ? ", selected" : ", available"}`}
                        aria-pressed={isSelected}
                        disabled={!seat.available || pending}
                        onClick={() => toggleSeat(seat)}
                      >
                        {seat.number}
                      </button>
                    );
                  })}
                </div>
                <span className="seat-row-label">{rowLabel}</span>
              </div>
            ))}
          </div>
        )}

        <div className="seat-legend" aria-label="Seat availability legend">
          <span><i className="seat-legend__swatch" /> Available</span>
          <span><i className="seat-legend__swatch seat-legend__swatch--selected" /> Selected</span>
          <span><i className="seat-legend__swatch seat-legend__swatch--unavailable" /> Booked</span>
        </div>

        <div className="seat-selection">
          <div>
            <strong>{selected.length ? selected.join(", ") : "No seats selected"}</strong>
            <span>{selected.length} of 10 seats · Select seats on the map</span>
          </div>
          <strong>{formatPrice(event.priceCents * selected.length)}</strong>
        </div>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="button button--wide" disabled={pending || loading || selected.length === 0} onClick={() => { void submit(); }}>
          {pending ? "Saving your seats..." : selected.length ? `Confirm ${selected.length} ${selected.length === 1 ? "seat" : "seats"}` : "Select your seats"}
        </button>
        <p className="fine-print">Seats update live as other guests book. No payment needed for this demo.</p>
      </section>
    </div>
  );
}
