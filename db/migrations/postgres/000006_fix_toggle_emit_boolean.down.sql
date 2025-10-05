-- Revert toggle strategy to emit an object with value field

UPDATE strategies
SET code = 'function process(context) {
    // Get the last output value, default to false
    const lastValue = context.lastOutputs.value || false;

    // Flip the boolean value
    const newValue = !Boolean(lastValue);

    context.emit({ value: newValue });
}'
WHERE id = 'toggle';
