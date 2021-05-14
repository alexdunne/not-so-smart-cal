import axios from "axios";

interface FetchEventRequestData {
  id: string;
}

interface FetchEventResponse {
  data: {
    event: {
      id: string;
      title: string;
      location: string | null;
      startsAt: string;
      endsAt: string;
    };
  };
}

interface CreateEventRequestData {
  title: string;
  location: string | null;
  startsAt: string;
  endsAt: string;
}

interface CreateEventResponse {
  data: {
    event: {
      id: string;
      title: string;
      location: string | null;
      startsAt: string;
      endsAt: string;
    };
  };
}

export const makeCalendarClient = () => {
  const client = axios.create({ baseURL: `${process.env.CALENDAR_SERVICE}` });

  return {
    fetchEvent: async (data: FetchEventRequestData) => {
      return client.get<FetchEventResponse>(`/event/${data.id}`);
    },
    createEvent: async (data: CreateEventRequestData) => {
      return client.post<CreateEventResponse>(`/event`, data);
    },
  };
};
