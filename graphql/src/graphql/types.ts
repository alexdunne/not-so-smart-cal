import axios from "axios";
import { objectType, queryField, asNexusMethod, nonNull, stringArg, inputObjectType } from "nexus";
import { DateTimeResolver } from "graphql-scalars";

export const DateTime = DateTimeResolver;

export const CalendarServiceDiagnosticsType = objectType({
  name: "CalendarServiceDiagnostics",
  definition(t) {
    t.string("version");
  },
});

export const DiagnosticsType = objectType({
  name: "Diagnostics",
  definition(t) {
    t.field("calendar", {
      type: CalendarServiceDiagnosticsType,
    });
  },
});

export const DiagnosticsQuery = queryField("diagnostics", {
  type: DiagnosticsType,
  async resolve() {
    const response = await axios.get(`${process.env.CALENDAR_SERVICE}`);

    return {
      calendar: response.data,
    };
  },
});

export const EventWeatherType = objectType({
  name: "EventWeather",
  definition(t) {
    t.string("type");
    t.string("description");
    t.string("temp");
  },
});

export const EventType = objectType({
  name: "Event",
  definition(t) {
    t.string("id");
    t.string("title");
    t.string("location");
    t.field("startsAt", {
      type: "DateTime",
    });
    t.field("endsAt", {
      type: "DateTime",
    });

    t.field("weather", {
      type: "EventWeather",
      resolve: async (parent, _, ctx) => {
        try {
          const response = await ctx.weatherServiceClient.fetchEventWeather({ id: parent.id });
          return response.data.data.weather;
        } catch (e) {
          return null;
        }
      },
    });
  },
});

export const EventInputType = inputObjectType({
  name: "EventInput",
  definition(t) {
    t.nonNull.string("id");
  },
});

export const EventQuery = queryField("event", {
  type: "Event",
  args: {
    input: nonNull(EventInputType),
  },
  async resolve(_, args, ctx) {
    const eventResponse = await ctx.calendarServiceClient.fetchEvent({ id: args.input.id });

    return eventResponse.data.data.event;
  },
});

export * from "./CreateEventMutation";
