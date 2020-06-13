import React from 'react';
import {shallow} from 'enzyme';

import {ActionButtonType} from 'utils/constants';

import ActionView from 'components/post_type/action_view/action_view';

describe('components/post_type/action_view/ActionView', () => {
    const samplePollId = 'samplepollid1';
    const baseProps = {
        post: {
            id: 'post_id',
            props: {
                poll_id: samplePollId,
            },
        },
        attachment: {
            actions: [
                {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                {id: 'action_id2', name: 'answer2', type: ActionButtonType.BUTTON},
                {id: 'action_id3', name: 'answer3', type: ActionButtonType.BUTTON},
                {id: 'addOption', name: 'Add option', type: ActionButtonType.BUTTON},
                {id: 'deletePoll', name: 'Delete Poll', type: ActionButtonType.MATTERPOLL_ADMIN_BUTTON},
                {id: 'endPoll', name: 'End Poll', type: ActionButtonType.MATTERPOLL_ADMIN_BUTTON},
            ],
        },
        pollMetadata: {
            samplepollid1: {
                voted_answers: ['answer1', 'answer2'],
                poll_id: samplePollId,
                user_id: 'user_id1',
                admin_permission: false,
                setting_public_add_option: false,
            },
        },
        siteUrl: 'http://localhost:8065',
        actions: {
            fetchPollMetadata: jest.fn(),
        },
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<ActionView {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with admin permission', () => {
        const newProps = {
            ...baseProps,
            pollMetadata: {
                samplepollid1: {
                    voted_answers: ['answer1', 'answer2'],
                    poll_id: samplePollId,
                    user_id: 'user_id1',
                    admin_permission: true,
                },
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without permission for adding options', () => {
        const newProps = {
            ...baseProps,
            pollMetadata: {
                samplepollid1: {
                    voted_answers: ['answer1', 'answer2'],
                    poll_id: samplePollId,
                    user_id: 'user_id1',
                    admin_permission: false,
                    setting_public_add_option: false,
                },
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without admin permission, with public-add-option', () => {
        const newProps = {
            ...baseProps,
            pollMetadata: {
                samplepollid1: {
                    voted_answers: ['answer1', 'answer2'],
                    poll_id: samplePollId,
                    user_id: 'user_id1',
                    admin_permission: false,
                    setting_public_add_option: true,
                },
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without any actions', () => {
        const newProps = {
            ...baseProps,
            attachment: {actions: []},
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with only button actions', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                actions: [
                    {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                    {id: 'action_id2', name: 'answer2', type: ActionButtonType.SELECT},
                ],
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with only non-empty aciton id and name', () => {
        const newProps = {
            ...baseProps,
            attachment: {
                actions: [
                    {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                    {id: 'action_id2', name: '', type: ActionButtonType.BUTTON},
                    {id: '', name: 'answer3', type: ActionButtonType.BUTTON},
                    {id: 'action_id4', name: 'answer4', type: ActionButtonType.BUTTON},
                ],
            },
        };
        const wrapper = shallow(<ActionView {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
