import { useState } from "react";
import { TicketCategoryAvailability } from "./booking.api";

export function useBookingState() {
  const [availability] = useState<TicketCategoryAvailability[]>([]);
  const [isReserving] = useState(false);

  const fetchAvailability = async () => {
    // Placeholder state loader
  };

  const reserve = async (_category: string) => {
    // Placeholder reservation trigger
  };

  return {
    availability,
    isReserving,
    fetchAvailability,
    reserve,
  };
}
