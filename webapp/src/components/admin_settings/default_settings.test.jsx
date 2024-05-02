import React from 'react';
import {shallow} from 'enzyme';

import DefaultSettings from '@/components/admin_settings/default_settings';

describe('components/admin_settings/DefaultSettings', () => {
    const baseProps = {
        id: 'test id',
        value: {
            anonymous: false,
            anonymousCreator: true,
            progress: false,
            publicAddOption: true,
        },
        label: 'test label',
        disabled: false,
        setByEnv: false,
        config: {},
        license: {},
        onChange: jest.fn(),
        registerSaveAction: jest.fn(),
        setSaveNeeded: jest.fn(),
        unRegisterSaveAction: jest.fn(),
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<DefaultSettings {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with all true options', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.value = {
            anonymous: true,
            anonymousCreator: true,
            progress: true,
            publicAddOption: true,
        };
        const wrapper = shallow(<DefaultSettings {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with all false options', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.value = {
            anonymous: false,
            anonymousCreator: false,
            progress: false,
            publicAddOption: false,
        };
        const wrapper = shallow(<DefaultSettings {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
