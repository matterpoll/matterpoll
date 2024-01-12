import React from 'react';
import {shallow} from 'enzyme';

import FieldsTable from '@/components/post_type/fields/fields_table';

describe('components/post_type/fields/FiledsTable', () => {
    const baseProps = {
        attachment: {
            fields: [
                {
                    title: 'title1',
                    value: 'value1',
                    short: false,
                },
            ],
        },
        options: {
            mentionHighlight: false,
            markdown: false,
        },
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<FieldsTable {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without any fields', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.attachment.fields = [];
        const wrapper = shallow(<FieldsTable {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with two fields', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                fields: [
                    {title: 'title1', value: 'value1', short: false},
                    {title: 'title2', value: 'value2', short: false},
                ],
            },
        };
        const wrapper = shallow(<FieldsTable {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with a short field', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                fields: [
                    {title: 'title1', value: 'value1', short: true},
                ],
            },
        };
        const wrapper = shallow(<FieldsTable {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with two short fields', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                fields: [
                    {title: 'title1', value: 'value1', short: true},
                    {title: 'title2', value: 'value2', short: true},
                ],
            },
        };
        const wrapper = shallow(<FieldsTable {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with three short fields', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                fields: [
                    {title: 'title1', value: 'value1', short: true},
                    {title: 'title2', value: 'value2', short: true},
                    {title: 'title3', value: 'value3', short: true},
                ],
            },
        };
        const wrapper = shallow(<FieldsTable {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
