import Joi from "@hapi/joi";
import { mutationField, inputObjectType, nonNull, arg } from "nexus";

const CreateEventInputSchema = Joi.object({
  title: Joi.string().required(),
  location: Joi.string(),
  startsAt: Joi.date().iso().required(),
  endsAt: Joi.date().iso().required(),
});

export const CreateEventInputType = inputObjectType({
  name: "CreateEventInput",
  definition(t) {
    t.nonNull.string("title"), t.string("location"), t.nonNull.field("startsAt", { type: "DateTime" });
    t.nonNull.field("endsAt", { type: "DateTime" });
  },
});

export const createEvent = mutationField("createEvent", {
  type: "Event",
  args: {
    input: nonNull(arg({ type: "CreateEventInput" })),
  },
  async resolve(_, { input }, ctx) {
    await CreateEventInputSchema.validateAsync(input);

    const response = await ctx.calendarServiceClient.createEvent(input);
    const event = response.data.data.event;

    return {
      id: event.id,
      title: event.title,
      location: event.location,
      startsAt: event.startsAt,
      endsAt: event.endsAt,
    };
  },
});
