import React from 'react';
import {shallow} from 'enzyme';

import DefaultSetting from '@/components/admin_settings/default_setting';

describe('components/admin_settings/DefaultSettings', () => {
    const baseProps = {
        name: 'test name',
        title: 'test title',
        label: 'test label',
        value: false,
        onChange: jest.fn(),
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<DefaultSetting {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with true option', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.value = true;
        const wrapper = shallow(<DefaultSetting {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
