import { useCallback, useEffect, useMemo, useState } from "react";
import AuthDialog, { type AuthMode } from "./components/AuthDialog";
import BookingDialog from "./components/BookingDialog";
import { SiteFooter, SiteHeader, Toast, type PageName } from "./components/SiteChrome";
import BookingsPage from "./pages/BookingsPage";
import DiscoverPage from "./pages/DiscoverPage";
import ProfilePage from "./pages/ProfilePage";
import { api } from "./api";
import type { Booking, EventListing, User } from "./types";

export default function App() {
  const [events, setEvents] = useState<EventListing[]>([]);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [user, setUser] = useState<User | null>(null);
  const [page, setPage] = useState<PageName>("discover");
  const [category, setCategory] = useState("All events");
  const [search, setSearch] = useState("");
  const [authMode, setAuthMode] = useState<AuthMode | null>(null);
  const [bookingEvent, setBookingEvent] = useState<EventListing | null>(null);
  const [pendingEvent, setPendingEvent] = useState<EventListing | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const loadEvents = useCallback(async () => {
    try {
      setEvents(await api.events());
      setError("");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not load events.");
    }
  }, []);

  const loadBookings = useCallback(async () => {
    try {
      setBookings(await api.bookings());
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not load your bookings.");
    }
  }, []);

  useEffect(() => {
    let active = true;
    async function initialize() {
      try {
        const [currentUser, currentEvents] = await Promise.all([api.me(), api.events()]);
        if (!active) return;
        setUser(currentUser);
        setEvents(currentEvents);
        setError("");
        if (currentUser) {
          const currentBookings = await api.bookings();
          if (active) setBookings(currentBookings);
        }
      } catch (reason) {
        if (active) setError(reason instanceof Error ? reason.message : "Could not load Seatflow.");
      } finally {
        if (active) setLoading(false);
      }
    }
    void initialize();
    return () => { active = false; };
  }, []);

  useEffect(() => {
    const stream = new EventSource("/api/availability/stream", { withCredentials: true });
    stream.addEventListener("availability", () => { void loadEvents(); });
    const refresh = window.setInterval(() => { void loadEvents(); }, 15000);
    return () => {
      stream.close();
      window.clearInterval(refresh);
    };
  }, [loadEvents]);

  useEffect(() => {
    if (user) void loadBookings();
    else setBookings([]);
  }, [user, loadBookings]);

  useEffect(() => {
    if (!notice) return;
    const timeout = window.setTimeout(() => setNotice(""), 3800);
    return () => window.clearTimeout(timeout);
  }, [notice]);

  const categories = useMemo(
    () => ["All events", ...Array.from(new Set(events.map((event) => event.category)))],
    [events],
  );

  function startBooking(event: EventListing) {
    if (!user) {
      setPendingEvent(event);
      setAuthMode("login");
      return;
    }
    setBookingEvent(event);
  }

  function handleAuthSuccess(nextUser: User) {
    setUser(nextUser);
    setAuthMode(null);
    setNotice(`You're in, ${nextUser.name.split(" ")[0]}. Good choice.`);
    if (pendingEvent) {
      setBookingEvent(pendingEvent);
      setPendingEvent(null);
    }
  }

  async function handleBooked() {
    setBookingEvent(null);
    await Promise.all([loadEvents(), loadBookings()]);
    setNotice("You're on the list. See you there!");
    setPage("bookings");
  }

  async function handleCancel(booking: Booking) {
    if (!window.confirm(`Cancel your booking for "${booking.title}"?`)) return;
    try {
      await api.cancel(booking.id);
      await Promise.all([loadEvents(), loadBookings()]);
      setNotice("Booking cancelled. Your seats are back on the list.");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not cancel this booking.");
    }
  }

  async function handleSignOut() {
    try {
      await api.signOut();
      setUser(null);
      setNotice("You've signed out. Come back soon.");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not sign out.");
    }
  }

  return (
    <div className="app-shell">
      <SiteHeader
        user={user}
        bookings={bookings}
        page={page}
        onNavigate={setPage}
        onSignIn={() => setAuthMode("login")}
        onSignUp={() => setAuthMode("register")}
      />
      <main>
        {page === "discover" ? (
          <DiscoverPage
            events={events}
            loading={loading}
            error={error}
            categories={categories}
            category={category}
            search={search}
            onCategoryChange={setCategory}
            onSearchChange={setSearch}
            onRetry={() => { void loadEvents(); }}
            onBook={startBooking}
          />
        ) : page === "bookings" ? (
          <BookingsPage
            bookings={bookings}
            error={error}
            onRetry={() => { void loadBookings(); }}
            onCancel={(booking) => { void handleCancel(booking); }}
            onDiscover={() => setPage("discover")}
          />
        ) : (
          <ProfilePage
            user={user}
            bookings={bookings}
            error={error}
            onSignIn={() => setAuthMode("login")}
            onSignUp={() => setAuthMode("register")}
            onSignOut={() => { void handleSignOut(); }}
            onBookings={() => setPage("bookings")}
            onDiscover={() => setPage("discover")}
          />
        )}
      </main>
      <SiteFooter onHome={() => setPage("discover")} />
      {notice && <Toast message={notice} onDismiss={() => setNotice("")} />}
      {authMode && (
        <AuthDialog
          mode={authMode}
          onClose={() => { setAuthMode(null); setPendingEvent(null); }}
          onModeChange={setAuthMode}
          onSuccess={handleAuthSuccess}
        />
      )}
      {bookingEvent && (
        <BookingDialog
          event={bookingEvent}
          onClose={() => setBookingEvent(null)}
          onBooked={() => { void handleBooked(); }}
        />
      )}
    </div>
  );
}
