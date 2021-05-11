import axios from "axios";

interface CreateEventRequestData {
  title: string;
  location: string | null;
  startsAt: string;
  endsAt: string;
}

interface CreateEventResponseData {
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

const makeCalendarClient = () => {
  const client = axios.create({ baseURL: `${process.env.CALENDAR_SERVICE}` });

  return {
    createEvent: async (data: CreateEventRequestData) => {
      const response = await client.post<CreateEventResponseData>(`/event`, data);

      return response.data.data.event;
    },
  };
};

export const context = {
  calendarServiceClient: makeCalendarClient(),
};

export type Context = typeof context;
