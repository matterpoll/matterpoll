import React from 'react';
import {shallow} from 'enzyme';

import PostType from 'components/post_type/post_type';

describe('components/post_type/PostType', () => {
    const baseProps = {
        post: {
            props: {
                attachments: [
                    {
                        author_name: 'sample_name',
                        title: 'sample_title',
                        text: 'sample_text',
                    },
                ],
            },
        },
        theme: {},
        options: {},
        postTypeComponentId: '',
    };

    test('should match snapshot', () => {
        const wrapper = shallow(<PostType {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot without any attachments', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.post.props.attachments = [];

        const wrapper = shallow(<PostType {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with two attachments', () => {
        const newProps = {
            ...baseProps,
        };
        newProps.post.props.attachments.push({
            author_name: 'sample_name',
            title: 'sample_title',
            text: 'sample_text',
        });

        const wrapper = shallow(<PostType {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
