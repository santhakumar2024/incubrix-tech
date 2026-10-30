import EventCard from "../components/EventCard";
import type { EventListing } from "../types";

export default function DiscoverPage({
  events,
  loading,
  error,
  categories,
  category,
  search,
  onCategoryChange,
  onSearchChange,
  onRetry,
  onBook,
}: {
  events: EventListing[];
  loading: boolean;
  error: string;
  categories: string[];
  category: string;
  search: string;
  onCategoryChange: (category: string) => void;
  onSearchChange: (search: string) => void;
  onRetry: () => void;
  onBook: (event: EventListing) => void;
}) {
  const visibleEvents = events.filter((event) => {
    const query = search.trim().toLowerCase();
    const matchesCategory = category === "All events" || event.category === category;
    const matchesSearch = !query || [event.title, event.venue, event.city, event.category].some((value) => value.toLowerCase().includes(query));
    return matchesCategory && matchesSearch;
  });

  return (
    <>
      <section className="hero">
        <div className="hero-copy">
          <span className="eyebrow"><span className="live-dot" /> THE GOOD STUFF IS HAPPENING</span>
          <h1>Make room for<br />a <em>good time.</em></h1>
          <p>Little plans. Big memories. Find the things happening around you and save a spot for what feels like you.</p>
          <div className="hero-proof"><span className="proof-star">✳</span><span>Made for going out, not scrolling past.</span></div>
        </div>
        <div className="hero-art" aria-hidden="true">
          <div className="hero-sun" />
          <div className="hero-ring hero-ring--outer" />
          <div className="hero-ring hero-ring--inner" />
          <div className="hero-ticket"><span>YOUR NIGHT,</span><strong>YOUR<br />WAY.</strong><i>GOOD THINGS INSIDE ↗</i></div>
          <span className="hero-side-note">NEW YORK · RIGHT NOW</span>
          <span className="hero-spark hero-spark--one">✳</span>
          <span className="hero-spark hero-spark--two">✳</span>
        </div>
        <div className="hero-bottom">
          <span>01 — FIND YOUR THING</span>
          <span>SCROLL TO EXPLORE <span aria-hidden="true">↓</span></span>
        </div>
      </section>

      <section className="discover-section" id="events">
        <div className="section-heading">
          <div>
            <span className="section-kicker">A LITTLE SOMETHING FOR EVERYONE</span>
            <h2>What's your kind of <em>night?</em></h2>
          </div>
          <label className="search-box">
            <span className="search-icon" aria-hidden="true">⌕</span>
            <input aria-label="Search events" placeholder="Try “live music” or “Brooklyn”" value={search} onChange={(event) => onSearchChange(event.target.value)} />
            <kbd>↵</kbd>
          </label>
        </div>
        <div className="filter-row" role="group" aria-label="Filter by event category">
          {categories.map((item) => (
            <button className={category === item ? "filter-chip filter-chip--active" : "filter-chip"} key={item} onClick={() => onCategoryChange(item)}>{item}</button>
          ))}
          <span className="event-count">{visibleEvents.length} {visibleEvents.length === 1 ? "good thing" : "good things"}</span>
        </div>
        {error && <div className="error-banner" role="alert"><span>{error}</span><button onClick={onRetry}>Try again</button></div>}
        {loading ? (
          <div className="loading-state"><span className="spinner" /> Finding your next favorite thing...</div>
        ) : visibleEvents.length > 0 ? (
          <div className="event-grid">
            {visibleEvents.map((event) => <EventCard key={event.id} event={event} onBook={onBook} />)}
          </div>
        ) : error ? null : (
          <div className="no-results">
            <span>✳</span><h3>No plans found.</h3>
            <p>Try a different search or category. Your thing is out there.</p>
            <button className="button button--ghost" onClick={() => { onSearchChange(""); onCategoryChange("All events"); }}>Clear filters</button>
          </div>
        )}
      </section>
    </>
  );
}
