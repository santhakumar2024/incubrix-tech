import type { Booking, EventListing, EventSeat, User } from "./types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
  });
  const payload = (await response.json().catch(() => ({}))) as T & {
    error?: string;
  };
  if (!response.ok) {
    throw new Error(payload.error ?? "Something went wrong. Please try again.");
  }
  return payload;
}

export const api = {
  async me() {
    return (await request<{ user: User | null }>("/api/me")).user;
  },
  async events() {
    return (await request<{ events: EventListing[] }>("/api/events")).events;
  },
  async seats(eventId: number) {
    return (await request<{ seats: EventSeat[] }>(`/api/events/${eventId}/seats`)).seats;
  },
  async bookings() {
    return (await request<{ bookings: Booking[] }>("/api/bookings")).bookings;
  },
  async signIn(email: string, password: string) {
    return (
      await request<{ user: User }>("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      })
    ).user;
  },
  async signUp(name: string, email: string, password: string) {
    return (
      await request<{ user: User }>("/api/auth/register", {
        method: "POST",
        body: JSON.stringify({ name, email, password }),
      })
    ).user;
  },
  async signOut() {
    await request("/api/auth/logout", { method: "POST" });
  },
  async book(eventId: number, seatLabels: string[]) {
    return request<{ booking: Booking }>("/api/bookings", {
      method: "POST",
      body: JSON.stringify({ eventId, seatLabels }),
    });
  },
  async cancel(bookingId: number) {
    return request(`/api/bookings/${bookingId}`, { method: "DELETE" });
  },
};
