import {render, screen} from '@testing-library/react';
import React from 'react';

import FieldsTable from '@/components/post_type/fields/fields_table';
import type {AttachmentField} from '@/types/poll';

describe('components/post_type/fields/FieldsTable', () => {
    const baseOptions = {
        mentionHighlight: false,
        markdown: false,
    };

    const renderFields = (fields: AttachmentField[]) => render(
        <FieldsTable
            attachment={{fields}}
            options={baseOptions}
        />,
    );

    // A long (non-short) field always occupies a table of its own; short fields pair up
    // two per table, so the table count is what encodes the layout.
    const tableCount = (container: HTMLElement) => container.querySelectorAll('table.attachment-fields').length;

    test('should render a table for a single long field', () => {
        const {container} = renderFields([{title: 'title1', value: 'value1', short: false}]);

        expect(tableCount(container)).toBe(1);
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(title1))')).toBeInTheDocument();
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(value1))')).toBeInTheDocument();
    });

    test('should keep the class names and column width the webapp styles fields by', () => {
        const {container} = renderFields([{title: 'title1', value: 'value1', short: false}]);

        // width moved from a `width='50%'` attribute to an inline style when React 19
        // dropped the attribute from <th>'s types; the rendered width must not change.
        expect(container.querySelector('th.attachment-field__caption')).toHaveStyle({width: '50%'});
        expect(container.querySelector('td.attachment-field')).toBeInTheDocument();
    });

    test('should not emit an empty table ahead of a long field', () => {
        const {container} = renderFields([{title: 'title1', value: 'value1', short: false}]);

        container.querySelectorAll('table.attachment-fields').forEach((table) => {
            expect(table.querySelector('th')).not.toBeNull();
        });
    });

    test('should not emit an empty table for a long field followed by a short one', () => {
        const {container} = renderFields([
            {title: 'title1', value: 'value1', short: false},
            {title: 'title2', value: 'value2', short: true},
        ]);

        expect(tableCount(container)).toBe(2);
        container.querySelectorAll('table.attachment-fields').forEach((table) => {
            expect(table.querySelector('th')).not.toBeNull();
        });
    });

    test('should render nothing without any fields', () => {
        const {container} = renderFields([]);

        expect(container).toBeEmptyDOMElement();
    });

    test('should give each of two long fields its own table', () => {
        const {container} = renderFields([
            {title: 'title1', value: 'value1', short: false},
            {title: 'title2', value: 'value2', short: false},
        ]);

        expect(tableCount(container)).toBe(2);
    });

    test('should render a single short field', () => {
        const {container} = renderFields([{title: 'title1', value: 'value1', short: true}]);

        expect(tableCount(container)).toBe(1);
        expect(container.querySelectorAll('th')).toHaveLength(1);
    });

    test('should pair two short fields into one table', () => {
        const {container} = renderFields([
            {title: 'title1', value: 'value1', short: true},
            {title: 'title2', value: 'value2', short: true},
        ]);

        expect(tableCount(container)).toBe(1);
        expect(container.querySelectorAll('th')).toHaveLength(2);
    });

    test('should wrap a third short field into a second table', () => {
        const {container} = renderFields([
            {title: 'title1', value: 'value1', short: true},
            {title: 'title2', value: 'value2', short: true},
            {title: 'title3', value: 'value3', short: true},
        ]);

        expect(tableCount(container)).toBe(2);
    });
});
