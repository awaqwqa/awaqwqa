$(document).ready(function(){

    //头部语言鼠标悬停显示多语言选项
    lang();
    function lang(){
        if($(window).innerWidth()>1200){
            var lang=$(".lang");
            lang.hover(function () {
                $(this).find(".lang_son").stop(true,false).slideDown();
            },function () {
                $(this).find(".lang_son").stop(true,false).slideUp();
            });
        }
    }

    //手机端导航的js
    //导航按钮点击三横变一×
    click_nav_menu();
    function click_nav_menu() {
        $(".nav_menu").click(function () {
            //导航按钮三横变一×
            $(this).stop(true,true).toggleClass("click_nav_menu");
            //手机导航的显示隐藏
            $(".nav_xiala").stop(true,false).slideToggle();
            //手机端二级导航打开的时候，导航滚动条滚动到底后，会带动后面的身体部分的滚动条滚动，目前安卓端有bug不知怎么解决
            // $("body").toggleClass("hide_body_scrolly");
            if ($(window).scrollTop() <50) {
                var head = $(".head");
                if(head.hasClass("head_down")){
                    head.removeClass("head_down");
                    head.find(".logo img").attr("src","img/logo.png");
                }else{
                    head.addClass("head_down");
                    head.find(".logo img").attr("src","img/logo_down.png");
                }

            }
        });
    }

    //ios端让头部nav的滚动条一直显示
    nav_scroll();
    function nav_scroll() {
        var nav_box=$(".nav_box");
        var nav=$(".nav");
        if($(window).innerWidth()<1200){
            if(nav_box.innerHeight()<nav.children().length*nav.find("li").innerHeight()){
                $(".nav_scroll").css({'display':'block'})
            }
        }
        // console.log($(window).innerWidth());
        // console.log(nav_box.height());
        // console.log(nav.children().length*nav.find("li").innerHeight());
    }

    //搜索
    search();
    function search() {
        var search_btn1=$(".search_btn_1");
        var search_close=$(".search_close");
        var search_son=$(".search_son");
        //搜索放大镜点击
        search_btn1.click(function () {
            $(this).stop(true,false).toggle();
            search_close.stop(true,false).toggle();
            search_son.stop(true,false).slideToggle();
        });
        //搜索关闭按钮点击
        search_close.click(function () {
            $(this).stop(true,false).toggle();
            search_btn1.stop(true,false).toggle();
            search_son.stop(true,false).slideToggle();
        });
    }


    //头部样式变换
    scroll_head();
    function scroll_head(){
        // console.log($(window).width());
        var head = $(".head");
            $(window).scroll(function () {
                if ($(window).scrollTop() >= 50) {
                    head.addClass("head_down");
                    head.find(".logo img").attr("src","img/logo_down.png");
                } else {
                    head.removeClass("head_down");
                    head.find(".logo img").attr("src","img/logo.png")
                }
            });
    }

    //暂停开始banner下的video
    play_stop_video();
    function play_stop_video() {
        var banner =$('.banner');
        var video = $("#video1");
        $(window).scroll(function () {

            if ($(window).scrollTop() >= banner.innerHeight()) {
                if(video.hasClass("play")){
                    video.removeClass('play');
                    video.addClass('pause');
                    video.trigger('pause');
                }
            }else{

                if(video.hasClass("pause")){
                    video.removeClass('pause');
                    video.addClass('play');
                    video.trigger('play');
                }
            }

        });
    }

    //team左滑动
    scroll_team();
    function scroll_team() {
        var scroll=$(".team_scroll");
        var img=$(".team_img");
        var img_box=$(".team_img_box");
        img.find("img").load(function () {
            var win_width=$(window).width();
            var img_width=img.find("img").width();
            //未防止发烧友超宽的显示器，把浏览器宽度除以图片宽度的倍数来增加图片，使图片始终可以滚动
            var num=win_width/img_width;
            if(num>1){
                Math.ceil(win_width/img_width);
                // console.log(num);
                for(var i=0;i<num;i++){
                    img.append(img.html());
                }
            }
            img_box.append(img_box.html());

            setInterval(function () {

                if(scroll.scrollLeft()>=img.width()){
                    scroll.scrollLeft(0);
                }else{
                    scroll.scrollLeft(scroll.scrollLeft()+1);
                }

            },100);

            // console.log(img.find("img").width());
            // console.log($(window).width());
            // console.log($(window).innerWidth());
        });
    }




    //index team
    index_team();
    function index_team() {
        if($(".team").length){
            // console.log(1);
            var Swiper_team = new Swiper('.swiper_team', {
                lazy: true,
                loop : true,
                autoplay: {
                    delay: 3000,//3秒切换一次
                },
                // slidesPerView: 12,
                centeredSlides: true,
                breakpoints: {
                    320: {
                        slidesPerView: 5,
                    },
                    768: {
                        slidesPerView: 10,
                    },
                    1024: {
                        slidesPerView: 12,
                    },
                },
                on: {
                    transitionStart: function(){
                        var active=$(".team .swiper-slide-active");
                        var active_img=active.find(".team_item img");
                        $(".swiper-slide").find(".team_item img").css({'height':'100%','z-index':'0','border-radius': '0'});


                        if($(window).innerWidth()>1200){
                            active_img.css({'height':'calc(100% + 110px)','z-index':'1000','border-radius': '5px'});
                            active.next().find(".team_item img").css({'height':'calc(100% + 80px)','z-index':'900','border-radius': '5px'});
                            active.next().next().find(".team_item img").css({'height':'calc(100% + 60px)','z-index':'800','border-radius': '5px'});
                            active.next().next().next().find(".team_item img").css({'height':'calc(100% + 40px)','z-index':'700','border-radius': '5px'});
                            active.next().next().next().next().find(".team_item img").css({'height':'calc(100% + 20px)','z-index':'600','border-radius': '5px'});
                            active.next().next().next().next().next().find(".team_item img").css({'height':'calc(100% + 15px)','z-index':'500','border-radius': '5px'});
                            active.next().next().next().next().next().next().find(".team_item img").css({'height':'calc(100% + 10px)','z-index':'400','border-radius': '5px'});
                            active.next().next().next().next().next().next().next().find(".team_item img").css({'height':'calc(100% + 5px)','z-index':'300','border-radius': '5px'});

                            active.prev().find(".team_item img").css({'height':'calc(100% + 80px)','z-index':'900','border-radius': '5px'});
                            active.prev().prev().find(".team_item img").css({'height':'calc(100% + 60px)','z-index':'800','border-radius': '5px'});
                            active.prev().prev().prev().find(".team_item img").css({'height':'calc(100% + 40px)','z-index':'700','border-radius': '5px'});
                            active.prev().prev().prev().prev().find(".team_item img").css({'height':'calc(100% + 20px)','z-index':'600','border-radius': '5px'});
                            active.prev().prev().prev().prev().prev().find(".team_item img").css({'height':'calc(100% + 15px)','z-index':'500','border-radius': '5px'});
                            active.prev().prev().prev().prev().prev().prev().find(".team_item img").css({'height':'calc(100% + 10px)','z-index':'400','border-radius': '5px'});
                            active.prev().prev().prev().prev().prev().prev().prev().find(".team_item img").css({'height':'calc(100% + 5px)','z-index':'300','border-radius': '5px'});

                        }else{
                            active_img.css({'height':'calc(100% + 50px)','z-index':'1000','border-radius': '5px'});
                            active.next().find(".team_item img").css({'height':'calc(100% + 30px)','z-index':'900','border-radius': '5px'});
                            active.next().next().find(".team_item img").css({'height':'calc(100% + 10px)','z-index':'800','border-radius': '5px'});

                            active.prev().find(".team_item img").css({'height':'calc(100% + 30px)','z-index':'900','border-radius': '5px'});
                            active.prev().prev().find(".team_item img").css({'height':'calc(100% + 10px)','z-index':'800','border-radius': '5px'});
                        }

                        $(".team_info").html(active.find(".team_item a").attr("data-info"));
                    },
                },
            });



            //鼠标覆盖停止自动切换
            Swiper_team.el.onmouseover = function(){
                Swiper_team.autoplay.stop();
            };

            //鼠标离开开始自动切换
            Swiper_team.el.onmouseout = function(){
                Swiper_team.autoplay.start();
            };
        }
    }

    
    
//    大事记页面的swiper
    var store_control = new Swiper('.store_control', {
        freeMode:true,
        watchSlidesVisibility: true,
        watchSlidesProgress: true,
        breakpoints: {
            320: {
                slidesPerView: 4,
            },
            768: {
                slidesPerView: 5,
            },
            1024: {
                slidesPerView: 6,
            },
        },

    });
    var store = new Swiper('.store_swiper', {
        navigation:{
            nextEl:'.swiper-button-next',
            prevEl:'.swiper-button-prev',
        },
        thumbs: {
            swiper: store_control
        }
    });

//    默认页面下的二级导航
    var nav_two = new Swiper('.swiper-nav_two', {
        slidesPerView:'auto',
        freeMode:true,
    });


    //大事记页面和base封面页二级导航的判断是否超出原宽度如果没超过设置居中，如果超过了设置左对齐
    center_or_left();
    function center_or_left() {
        var nav=$(".nav_two .head_container");
        var nav_slides=$(".nav_two .head_container .swiper-slide");
        // console.log($(".nav_two .container").innerWidth());
        // console.log($(".nav_two .container .swiper-slide").length);
        // console.log($(".nav_two .container .swiper-slide").innerWidth());
        if(nav.innerWidth()>nav_slides.length*nav_slides.innerWidth()){
            nav.find(".swiper-wrapper").css({'justify-content':'center'})
        }

        var store=$(".store_control");
        var store_slides=$(".store_control .swiper-slide");
        // console.log(store.innerWidth());
        // console.log(store_slides.length);
        // console.log(store_slides.innerWidth());
        // console.log(store_slides.length*store_slides.innerWidth());
        if(store.innerWidth()>store_slides.length*store_slides.innerWidth()){
            store.find(".swiper-wrapper").css({'justify-content':'center'})
        }

    }

//封面下按钮的点击
    list_click();
    function list_click() {
        $("#list_tab li").click(function () {
            var index=$(this).index();
            $(this).parents().find(".list_item").removeClass("show_list_item").eq(index).addClass("show_list_item");
            $(this).addClass("list_tab_active").siblings().removeClass("list_tab_active");
        });
    }

//    荣誉等页面下的滚动
    list_item_swiper();
    function list_item_swiper() {
        var list_item1 = new Swiper('.list_item_swiper1', {
            slidesPerColumn:2,
            spaceBetween:20,
            breakpoints: {
                320: {
                    slidesPerView: 2,
                },
                768: {
                    slidesPerView: 2,
                },
                1024: {
                    slidesPerView: 3,
                },
            },
            navigation:{
                nextEl:'.swiper-button-next1',
                prevEl:'.swiper-button-prev1',
            },
        });
        var list_item2 = new Swiper('.list_item_swiper2', {
            slidesPerColumn:2,
            spaceBetween:20,
            breakpoints: {
                320: {
                    slidesPerView: 2,
                },
                768: {
                    slidesPerView: 2,
                },
                1024: {
                    slidesPerView: 3,
                },
            },
            navigation:{
                nextEl:'.swiper-button-next2',
                prevEl:'.swiper-button-prev2',
            },
        });
        var list_item3 = new Swiper('.list_item_swiper3', {
            slidesPerColumn:2,
            spaceBetween:20,
            breakpoints: {
                320: {
                    slidesPerView: 2,
                },
                768: {
                    slidesPerView: 2,
                },
                1024: {
                    slidesPerView: 3,
                },
            },
            navigation:{
                nextEl:'.swiper-button-next3',
                prevEl:'.swiper-button-prev3',
            },
        });
        var list_item4 = new Swiper('.list_item_swiper4', {
            slidesPerColumn:2,
            spaceBetween:20,
            breakpoints: {
                320: {
                    slidesPerView: 2,
                },
                768: {
                    slidesPerView: 2,
                },
                1024: {
                    slidesPerView: 3,
                },
            },
            navigation:{
                nextEl:'.swiper-button-next4',
                prevEl:'.swiper-button-prev4',
            },
        });
    }

//    伙伴页面的swiper
    var list_huoban = new Swiper('.list_huoban_swiper', {
        loop:true,
        autoplay: {
            delay: 3000,//3秒切换一次
        },
        spaceBetween:20,
        breakpoints: {
            320: {
                slidesPerView: 2,
            },
            768: {
                slidesPerView: 2,
            },
            1024: {
                slidesPerView: 3,
            },
        },
    });

//    业务页面的人员swiper
    var business_team = new Swiper('.business_team_swiper', {
        loop:true,
        autoplay: {
            delay: 3000,//3秒切换一次
        },
        spaceBetween:20,
        breakpoints: {
            320: {
                slidesPerView: 3,
            },
            768: {
                slidesPerView: 6,
            },
            1024: {
                slidesPerView: 6,
            },
        },
    });



// about视频点击按钮
    about_play_video();
    function about_play_video() {
        var video=$("#about_video");
        $(".about_button span").click(function () {
            video.attr("src",video.attr("data-src",));
            video.addClass("about_play");
            video.trigger('play');
        });
    }


    //列表页下面的单图图片高度设置
    list_article_img_height();
    function list_article_img_height(){
        var img=$(".info_img");
        // console.log(img.height());
        if($(window).innerWidth()>768) {
            if (img.length) {
                img.each(function () {
                    $(this).find(".info_img_box").css({'height': $(this).height()})
                })
            }
        }
    }

//    人员页面的右边高度设置
    renyuan_height();
    function renyuan_height() {
        var renyuan=$(".renyuan");
        if($(window).innerWidth()>768) {
            if (renyuan.length) {
                if(renyuan.find(".renyuan_left img").load()) {
                    // console.log(renyuan.find(".renyuan_left img").innerHeight());
                    // console.log(renyuan.find(".renyuan_right").innerHeight());
                    // renyuan.find(".renyuan_left img").innerHeight();
                    if(renyuan.find(".renyuan_left img").innerHeight()>renyuan.find(".renyuan_right").innerHeight()){
                        // var top=(renyuan.find(".renyuan_left img").innerHeight()-renyuan.find(".renyuan_right").innerHeight())/2;
                        // renyuan.find(".renyuan_right").css({'margin-top':top})
                    }else{
                        renyuan.find(".renyuan_right").css({'height':renyuan.find(".renyuan_left img").innerHeight()-renyuan.find(".renyuan_title").innerHeight()-renyuan.find(".renyuan_contact").innerHeight()})
                    }
                }
            }
        }
    }


    //图片懒加载代码
    lazy();
    function lazy(){
         start();
         $(window).on('scroll', function(){
             start()
         });
    }
    
    function start() {
        var $lazy = $('.lazy');
            //.not('[data-isLoaded]')选中已加载的图片不需要重新加载
            $lazy.not('[data-isLoaded]').each(function(){
                var $node = $(this);
                if( isShow($node) ){
                    loadImg($node);
                    $node.load(function(){
                        $node.parent().css('background','none');
                    });
                }
            })
    }

    //判断一个元素是不是出现在窗口(视野)
    function isShow($node){
        if(parseInt($node.offset().top) <= parseInt(window.innerHeight + $(window).scrollTop())){
            console.log('当前元素位置'+$node.offset().top);
            console.log('当前窗口位置'+(window.innerHeight + $(window).scrollTop()));
            console.log('当前元素在显示区域内');
            return true;
        }else{
            // console.log('当前元素位置'+$node.offset().top);
            // console.log('当前窗口位置'+(window.innerHeight + $(window).scrollTop()));
            //console.log('当前元素不在显示区域内');
            return false;
        }
    }
    //加载图片
    function loadImg($node){
        //.attr(值)
        //.attr(属性名称,值)
        $node.css({'background':'url('+$node.attr("data-src")+')'})
        $node.find(".loading").css({'display':'none'});//把data-src的值 赋值给src
        $node.attr('data-isLoaded', 1)
    }

    
//    job页面的点击
    job_click();
    function job_click() {
        var button= $(".job_button");
        button.click(function () {
            $(this).parent().find(".job_content").slideToggle();
            $(this).parent().siblings().find(".job_content").slideUp();
        })

        //
        // button.each(function () {
        //     $(this).click(function () {
        //         $(this).parents().siblings().find(".job_content").slideUp();
        //         $(this).parents().find(".job_content").slideDown();
        //     })
        // })


    }

});




window.onload=function(){
    setTimeout(function () {
        var video = $("#video1");
        video.attr('src',video.attr("data-src"));
        video.trigger('play');
        video.addClass('play');
        $("body").addClass("ok");
    },500);
}