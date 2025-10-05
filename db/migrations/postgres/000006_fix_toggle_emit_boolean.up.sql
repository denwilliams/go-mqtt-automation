-- Update toggle strategy to emit a simple boolean instead of an object

UPDATE strategies
SET code = 'function process(context) {
    // Get the last output value, default to false
    const lastValue = context.lastOutputs || false;

    // Flip the boolean value
    const newValue = !Boolean(lastValue);

    context.emit(newValue);
}'
WHERE id = 'toggle';
