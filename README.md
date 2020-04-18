# simple-text

## Use Case
* You're a church group (or similar) that wants to occasionally send out text messages to the group.
* You don't care about replies to the messages.
  * Using Twilio Functions you can auto-reply to the messages indicating that the number doesn't accept incoming messages but folks can direct their queries to Joe at 555-555-5555.

Here's a first cut at a Twilio Function:
```py
/*
   After you have deployed your Function, head to your phone number and configure the inbound SMS handler to this Function
*/
exports.handler = function(context, event, callback) {
    let twiml = new Twilio.twiml.MessagingResponse();

    const body = event.Body ? event.Body.toLowerCase() : null;
    if (body == 'stop') {
        callback();
        return
    }
    // TODO: DANGER - we should only send this once per number within x days or we could cause a bots to text each other out of control!
    twiml.message("This number doesn't accept incoming messages. If you need assistance please contact the ward clerk at 555-555-5555.\n\nReply STOP to unsubscribe.");
    callback(null, twiml);
};
```