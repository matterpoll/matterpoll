import React from 'react';
import {render, screen} from '@testing-library/react';

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
    // two per table, so the table count is what encodes the layout. The component flushes
    // the accumulator before starting a long field, which leaves an empty leading table --
    // long-standing behaviour, so only tables that actually hold a field are counted here.
    const tableCount = (container: HTMLElement) =>
        [...container.querySelectorAll('table.attachment-fields')].filter((t) => t.querySelector('th')).length;

    test('should render a table for a single long field', () => {
        const {container, asFragment} = renderFields([{title: 'title1', value: 'value1', short: false}]);

        expect(tableCount(container)).toBe(1);
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(title1))')).toBeInTheDocument();
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(value1))')).toBeInTheDocument();
        expect(asFragment()).toMatchSnapshot();
    });

    test('should emit an empty leading table before a long field', () => {
        // Pinned deliberately: this is existing behaviour, not something the
        // React 19 migration introduced.
        const {container} = renderFields([{title: 'title1', value: 'value1', short: false}]);

        const tables = container.querySelectorAll('table.attachment-fields');
        expect(tables).toHaveLength(2);
        expect(tables[0].querySelector('th')).toBeNull();
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
