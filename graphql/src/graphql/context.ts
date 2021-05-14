import { makeCalendarClient } from "../api/calendarClient";
import { makeWeatherClient } from "../api/weatherClient";

export const context = {
  calendarServiceClient: makeCalendarClient(),
  weatherServiceClient: makeWeatherClient(),
};

export type Context = typeof context;
