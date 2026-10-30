export interface User {
  id: number;
  name: string;
  email: string;
}

export interface EventListing {
  id: number;
  slug: string;
  title: string;
  category: string;
  venue: string;
  city: string;
  startsAt: string;
  description: string;
  totalCapacity: number;
  bookedQuantity: number;
  available: number;
  priceCents: number;
  accent: string;
}

export interface Booking {
  id: number;
  eventId: number;
  quantity: number;
  seatLabels: string[];
  status: "confirmed" | "cancelled";
  createdAt: string;
  title: string;
  category: string;
  venue: string;
  city: string;
  startsAt: string;
  priceCents: number;
}

export interface EventSeat {
  label: string;
  row: string;
  number: number;
  available: boolean;
}
