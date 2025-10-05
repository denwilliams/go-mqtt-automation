-- Add comparison strategies: less_than, greater_than, less_than_or_equal, greater_than_or_equal

INSERT INTO strategies (id, name, description, code) VALUES
('less_than', 'Less Than', 'Returns true if the input value is less than the threshold parameter.', 'function process(context) {
    const value = Number(context.inputs.value);
    const threshold = Number(context.parameters.threshold || 0);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value < threshold);
}'),
('greater_than', 'Greater Than', 'Returns true if the input value is greater than the threshold parameter.', 'function process(context) {
    const value = Number(context.inputs.value);
    const threshold = Number(context.parameters.threshold || 0);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value > threshold);
}'),
('less_than_or_equal', 'Less Than or Equal', 'Returns true if the input value is less than or equal to the threshold parameter.', 'function process(context) {
    const value = Number(context.inputs.value);
    const threshold = Number(context.parameters.threshold || 0);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value <= threshold);
}'),
('greater_than_or_equal', 'Greater Than or Equal', 'Returns true if the input value is greater than or equal to the threshold parameter.', 'function process(context) {
    const value = Number(context.inputs.value);
    const threshold = Number(context.parameters.threshold || 0);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value >= threshold);
}');
